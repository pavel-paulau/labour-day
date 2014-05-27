package main

import (
	"encoding/json"
	"log"
	"sort"

	"github.com/couchbaselabs/go-couchbase"
)

var ddocs = map[string]string{
	"jenkins": `{
		"views": {
			"data_by_build": {
				"map": "function (doc, meta) {emit(doc.build, [doc.failCount, doc.totalCount, doc.os, doc.priority, doc.component]);}"
			}
		}
	}`,
}

type DataSource struct {
	CouchbaseAddress string
	Release          string
}

func (ds *DataSource) GetBucket(bucket string) *couchbase.Bucket {
	client, _ := couchbase.Connect(ds.CouchbaseAddress)
	pool, _ := client.GetPool("default")

	b, err := pool.GetBucket(bucket)
	if err != nil {
		log.Fatalf("Error reading bucket:  %v", err)
	}
	return b
}

func (ds *DataSource) QueryView(b *couchbase.Bucket, ddoc, view string,
	params map[string]interface{}) []couchbase.ViewRow {
	params["stale"] = "false"
	vr, err := b.View(ddoc, view, params)
	if err != nil {
		ds.installDDoc(ddoc)
	}
	return vr.Rows
}

func (ds *DataSource) installDDoc(ddoc string) {
	b := ds.GetBucket(ddoc) // bucket name == ddoc name
	err := b.PutDDoc(ddoc, ddocs[ddoc])
	if err != nil {
		log.Fatalf("%v", err)
	}
}

var TIMELINE_SIZE = 40

var VIEW = map[string]int{
	"fail_count":  0,
	"total_count": 1,
	"by_platform": 2,
	"by_priority": 3,
	"by_category": 4,
}

type MapBuild struct {
	Passed   float64
	Failed   float64
	Category string
	Platform string
	Priority string
}

type Breakdown struct {
	Passed float64
	Failed float64
}

type ReduceBuild struct {
	Version    string
	AbsPassed  float64
	AbsFailed  float64
	RelPassed  float64
	RelFailed  float64
	ByCategory map[string]Breakdown
	ByPlatform map[string]Breakdown
	ByPriority map[string]Breakdown
}

type FullSet struct {
	ByPlatform map[string]Breakdown
	ByPriority map[string]Breakdown
}

func appendIfUnique(slice []string, s string) []string {
	for i := range slice {
		if slice[i] == s {
			return slice
		}
	}
	return append(slice, s)
}

func posInSlice(slice []string, s string) int {
	for i := range slice {
		if slice[i] == s {
			return i
		}
	}
	return -1
}

func (ds *DataSource) GetTimeline() []byte {
	b := ds.GetBucket("jenkins")
	params := map[string]interface{}{"startkey": ds.Release}
	rows := ds.QueryView(b, "jenkins", "data_by_build", params)

	/***************** MAP *****************/
	mapBuilds := map[string][]MapBuild{}
	for _, row := range rows {
		version := row.Key.(string)
		failed, ok := row.Value.([]interface{})[VIEW["fail_count"]].(float64)
		if !ok {
			continue
		}
		total, ok := row.Value.([]interface{})[VIEW["total_count"]].(float64)
		if !ok {
			continue
		}
		category := row.Value.([]interface{})[VIEW["by_category"]].(string)
		platform := row.Value.([]interface{})[VIEW["by_platform"]].(string)
		priority := row.Value.([]interface{})[VIEW["by_priority"]].(string)

		mapBuilds[version] = append(mapBuilds[version], MapBuild{
			total - failed,
			failed,
			category,
			platform,
			priority,
		})
	}

	/***************** REDUCE *****************/
	versions := []string{}
	for version, _ := range mapBuilds {
		versions = append(versions, version)
	}
	sort.Strings(versions)

	fullSet := map[string]FullSet{}

	allCategories := []string{}

	skip := len(versions) - TIMELINE_SIZE
	reduceBuilds := []ReduceBuild{}
	for _, version := range versions[skip:] {
		reduce := ReduceBuild{}
		reduce.Version = version
		reduce.ByCategory = map[string]Breakdown{}
		reduce.ByPlatform = map[string]Breakdown{}
		reduce.ByPriority = map[string]Breakdown{}
		currCategories := []string{}
		for _, build := range mapBuilds[version] {
			// Totals
			reduce.AbsPassed += build.Passed
			reduce.AbsFailed -= build.Failed

			// By Category
			if _, ok := reduce.ByCategory[build.Category]; ok {
				passed := reduce.ByCategory[build.Category].Passed + build.Passed
				failed := reduce.ByCategory[build.Category].Failed + build.Failed
				reduce.ByCategory[build.Category] = Breakdown{passed, failed}
			} else {
				reduce.ByCategory[build.Category] = Breakdown{build.Passed, build.Failed}
			}
			allCategories = appendIfUnique(allCategories, build.Category)

			// By Platform
			if _, ok := reduce.ByPlatform[build.Platform]; ok {
				passed := reduce.ByPlatform[build.Platform].Passed + build.Passed
				failed := reduce.ByPlatform[build.Platform].Failed + build.Failed
				reduce.ByPlatform[build.Platform] = Breakdown{passed, failed}
			} else {
				reduce.ByPlatform[build.Platform] = Breakdown{build.Passed, build.Failed}
			}

			// By Priority
			if _, ok := reduce.ByPriority[build.Priority]; ok {
				passed := reduce.ByPriority[build.Priority].Passed + build.Passed
				failed := reduce.ByPriority[build.Priority].Failed + build.Failed
				reduce.ByPriority[build.Priority] = Breakdown{passed, failed}
			} else {
				reduce.ByPriority[build.Priority] = Breakdown{build.Passed, build.Failed}
			}

			// Full Set
			if posInSlice(currCategories, build.Category) == -1 {
				byPlatform := map[string]Breakdown{
					build.Platform: Breakdown{build.Passed, build.Failed},
				}
				byPriority := map[string]Breakdown{
					build.Priority: Breakdown{build.Passed, build.Failed},
				}
				fullSet[build.Category] = FullSet{byPlatform, byPriority}
			} else {
				// By Platform
				passed := build.Passed + fullSet[build.Category].ByPlatform[build.Platform].Passed
				failed := build.Failed + fullSet[build.Category].ByPlatform[build.Platform].Failed
				fullSet[build.Category].ByPlatform[build.Platform] = Breakdown{passed, failed}

				// By Priority
				passed = build.Passed + fullSet[build.Category].ByPriority[build.Priority].Passed
				failed = build.Failed + fullSet[build.Category].ByPriority[build.Priority].Failed
				fullSet[build.Category].ByPriority[build.Priority] = Breakdown{passed, failed}
			}

			currCategories = appendIfUnique(currCategories, build.Category)
		}

		/***************** BACKFILL *****************/
		for _, category := range allCategories {
			totalPassed := float64(0)
			totalFailed := float64(0)
			if _, ok := reduce.ByCategory[category]; !ok {
				for platform, breakdown := range fullSet[category].ByPlatform {
					if _, ok := reduce.ByPlatform[platform]; ok {
						passed := reduce.ByPlatform[platform].Passed + breakdown.Passed
						failed := reduce.ByPlatform[platform].Failed + breakdown.Failed
						reduce.ByPlatform[platform] = Breakdown{passed, failed}
					} else {
						reduce.ByPlatform[platform] = Breakdown{breakdown.Passed, breakdown.Failed}
					}
					totalPassed += breakdown.Passed
					totalFailed += breakdown.Failed
				}

				for priority, breakdown := range fullSet[category].ByPriority {
					if _, ok := reduce.ByPriority[priority]; ok {
						passed := reduce.ByPriority[priority].Passed + breakdown.Passed
						failed := reduce.ByPriority[priority].Failed + breakdown.Failed
						reduce.ByPriority[priority] = Breakdown{passed, failed}
					} else {
						reduce.ByPriority[priority] = Breakdown{breakdown.Passed, breakdown.Failed}
					}
				}
				reduce.AbsPassed += totalPassed
				reduce.AbsFailed -= totalFailed
				reduce.ByCategory[category] = Breakdown{totalPassed, totalFailed}
			}
		}

		total := reduce.AbsPassed - reduce.AbsFailed
		reduce.RelPassed = 100.0 * reduce.AbsPassed / total
		reduce.RelFailed = -100.0 * reduce.AbsFailed / total
		reduceBuilds = append(reduceBuilds, reduce)
	}

	j, _ := json.Marshal(reduceBuilds)
	return j
}

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
	map_builds := map[string][]MapBuild{}
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

		map_builds[version] = append(map_builds[version], MapBuild{
			total - failed,
			failed,
			category,
			platform,
			priority,
		})
	}

	/***************** REDUCE *****************/
	versions := []string{}
	for version, _ := range map_builds {
		versions = append(versions, version)
	}
	sort.Strings(versions)

	full_set := map[string]FullSet{}

	all_categories := []string{}

	skip := len(versions) - TIMELINE_SIZE
	reduce_builds := []ReduceBuild{}
	for _, version := range versions[skip:] {
		reduce_build := ReduceBuild{}
		reduce_build.Version = version
		reduce_build.ByCategory = map[string]Breakdown{}
		reduce_build.ByPlatform = map[string]Breakdown{}
		reduce_build.ByPriority = map[string]Breakdown{}
		this_categories := []string{}
		for _, build := range map_builds[version] {
			// Totals
			reduce_build.AbsPassed += build.Passed
			reduce_build.AbsFailed -= build.Failed

			// By Category
			if _, ok := reduce_build.ByCategory[build.Category]; ok {
				passed := reduce_build.ByCategory[build.Category].Passed + build.Passed
				failed := reduce_build.ByCategory[build.Category].Failed + build.Failed
				reduce_build.ByCategory[build.Category] = Breakdown{passed, failed}
			} else {
				reduce_build.ByCategory[build.Category] = Breakdown{build.Passed, build.Failed}
			}
			all_categories = appendIfUnique(all_categories, build.Category)

			// By Platform
			if _, ok := reduce_build.ByPlatform[build.Platform]; ok {
				passed := reduce_build.ByPlatform[build.Platform].Passed + build.Passed
				failed := reduce_build.ByPlatform[build.Platform].Failed + build.Failed
				reduce_build.ByPlatform[build.Platform] = Breakdown{passed, failed}
			} else {
				reduce_build.ByPlatform[build.Platform] = Breakdown{build.Passed, build.Failed}
			}

			// By Priority
			if _, ok := reduce_build.ByPriority[build.Priority]; ok {
				passed := reduce_build.ByPriority[build.Priority].Passed + build.Passed
				failed := reduce_build.ByPriority[build.Priority].Failed + build.Failed
				reduce_build.ByPriority[build.Priority] = Breakdown{passed, failed}
			} else {
				reduce_build.ByPriority[build.Priority] = Breakdown{build.Passed, build.Failed}
			}

			// Full Set
			if posInSlice(this_categories, build.Category) == -1 {
				by_platform := map[string]Breakdown{
					build.Platform: Breakdown{build.Passed, build.Failed},
				}
				by_priority := map[string]Breakdown{
					build.Priority: Breakdown{build.Passed, build.Failed},
				}
				full_set[build.Category] = FullSet{by_platform, by_priority}
			} else {
				// By Platform
				passed := build.Passed + full_set[build.Category].ByPlatform[build.Platform].Passed
				failed := build.Failed + full_set[build.Category].ByPlatform[build.Platform].Failed
				full_set[build.Category].ByPlatform[build.Platform] = Breakdown{passed, failed}

				// By Priority
				passed = build.Passed + full_set[build.Category].ByPriority[build.Priority].Passed
				failed = build.Failed + full_set[build.Category].ByPriority[build.Priority].Failed
				full_set[build.Category].ByPriority[build.Priority] = Breakdown{passed, failed}
			}

			this_categories = appendIfUnique(this_categories, build.Category)
		}

		/***************** BACKFILL *****************/
		for _, category := range all_categories {
			total_passed := float64(0)
			total_failed := float64(0)
			if _, ok := reduce_build.ByCategory[category]; !ok {
				for platform, breakdown := range full_set[category].ByPlatform {
					if _, ok := reduce_build.ByPlatform[platform]; ok {
						passed := reduce_build.ByPlatform[platform].Passed + breakdown.Passed
						failed := reduce_build.ByPlatform[platform].Failed + breakdown.Failed
						reduce_build.ByPlatform[platform] = Breakdown{passed, failed}
					} else {
						reduce_build.ByPlatform[platform] = Breakdown{breakdown.Passed, breakdown.Failed}
					}
					total_passed += breakdown.Passed
					total_failed += breakdown.Failed
				}

				for priority, breakdown := range full_set[category].ByPriority {
					if _, ok := reduce_build.ByPriority[priority]; ok {
						passed := reduce_build.ByPriority[priority].Passed + breakdown.Passed
						failed := reduce_build.ByPriority[priority].Failed + breakdown.Failed
						reduce_build.ByPriority[priority] = Breakdown{passed, failed}
					} else {
						reduce_build.ByPriority[priority] = Breakdown{breakdown.Passed, breakdown.Failed}
					}
				}
				reduce_build.AbsPassed += total_passed
				reduce_build.AbsFailed -= total_failed
				reduce_build.ByCategory[category] = Breakdown{total_passed, total_failed}
			}
		}

		total := reduce_build.AbsPassed - reduce_build.AbsFailed
		reduce_build.RelPassed = 100.0 * reduce_build.AbsPassed / total
		reduce_build.RelFailed = -100.0 * reduce_build.AbsFailed / total
		reduce_builds = append(reduce_builds, reduce_build)
	}

	j, _ := json.Marshal(reduce_builds)
	return j
}

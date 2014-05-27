/*jshint jquery: true, browser: true*/
/*global d3: true*/

/*
 * http://cmaurer.github.io/angularjs-nvd3-directives/multi.bar.chart.html
 * http://cmaurer.github.io/angularjs-nvd3-directives/pie.chart.html
 */
function Timeline($scope, $http) {
	"use strict";

	$http.get('/timeline').success(function(data) {
		$scope.timelineData = data;
		$scope.timelineRelData = [{
			"key": "Passed, %",
			"values": []
		}, {
			"key": "Failed, %",
			"values": []
		}];

		$scope.timelineAbsData = [{
			"key": "Passed",
			"values": []
		}, {
			"key": "Failed",
			"values": []
		}];

		data.forEach(function(build) {
			$scope.timelineRelData[0].values.push([build.Version, build.RelPassed]);
			$scope.timelineRelData[1].values.push([build.Version, build.RelFailed]);
			$scope.timelineAbsData[0].values.push([build.Version, build.AbsPassed]);
			$scope.timelineAbsData[1].values.push([build.Version, build.AbsFailed]);
		});

		updateBreakDown(data.length - 1);
	});

	var format = d3.format('f');
	$scope.yAxisTickFormatFunction = function(){
		return function(d) {
			return format(Math.abs(d));
		};
	};

	$scope.relToolTipContentFunction = function() {
		return function(key, build, num) {
			return '<h4>' + num + '% Tests ' + key.replace(', %', '') + '</h4>' +
				'<p>Build ' + build + '</p>';
		};
	};

	$scope.absToolTipContentFunction = function() {
		return function(key, build, num, data) {
			var total = $scope.timelineData[data.pointIndex].AbsPassed -
				$scope.timelineData[data.pointIndex].AbsFailed;
			return '<h4>' + num + ' of ' + total + ' Tests ' + key + '</h4>' +
				'<p>Build ' + build + '</p>';
		};
	};

	$scope.$on('barClick', function(event, data) {
		updateBreakDown(data.pointIndex);
	});

	$scope.xFunction = function(){
		return function(d){ return d.key; };
	};

	$scope.yFunction = function(){
		return function(d){ return d.value; };
	};

	var updateBreakDown = function(seq_id) {
		$scope.build = $scope.timelineData[seq_id].Version;

		/****************************** ByPlatform ******************************/
		var data = $scope.timelineData[seq_id].ByPlatform;
		$scope.byPlatform = {};
		Object.keys(data).forEach(function(breakdown) {
			$scope.byPlatform[breakdown] = [{
				"key": "Passed",
				"value": data[breakdown].Passed
			}, {
				"key": "Failed",
				"value": data[breakdown].Failed
			}];
		});
		$scope.numPlatforms = Object.keys($scope.byPlatform).length;
		$scope.platformWidth = screen.width * 0.95 * 0.3 /$scope.numPlatforms;

		/****************************** ByPriority ******************************/
		data = $scope.timelineData[seq_id].ByPriority;
		$scope.byPriority = {};
		Object.keys(data).forEach(function(breakdown) {
			$scope.byPriority[breakdown] = [{
				"key": "Passed",
				"value": data[breakdown].Passed
			}, {
				"key": "Failed",
				"value": data[breakdown].Failed
			}];
		});
		$scope.numPriorities = Object.keys($scope.byPriority).length;
		$scope.priorityWidth = screen.width * 0.95 * 0.3 / $scope.numPriorities;

		/****************************** ByCategory ******************************/
		data = $scope.timelineData[seq_id].ByCategory;
		$scope.byCategory = {};
		Object.keys(data).forEach(function(breakdown) {
			$scope.byCategory[breakdown] = [{
				"key": "Passed",
				"value": data[breakdown].Passed
			}, {
				"key": "Failed",
				"value": data[breakdown].Failed
			}];
		});
		$scope.numCategories = Object.keys($scope.byCategory).length;
		$scope.categoryWidth = screen.width * 0.95 * 0.4 / $scope.numCategories;

		if(!$scope.$$phase) {
			$scope.$apply();
		}
	};

	$scope.breakdownToolTipContentFunction = function() {
		return function(status, num) {
			return '<h4>' + parseInt(num, 10) + ' Tests ' + status + '</h4>' +
				'<p>Build ' + $scope.build + '</p>';
		};
	};
}

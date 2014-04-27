/*jshint jquery: true, browser: true*/
/*global d3: true*/

/*
 * http://cmaurer.github.io/angularjs-nvd3-directives/multi.bar.chart.html
 * http://cmaurer.github.io/angularjs-nvd3-directives/pie.chart.html
 */
function Timeline($scope, $http) {
	"use strict";

	$http.get('/rel_timeline').success(function(data) {
		$scope.timelineRelData = data;
		var latest_build = data[0]["values"][data[0]["values"].length - 1][0]
		updateBreakDown(latest_build);
	});

	$http.get('/abs_timeline').success(function(data) {
		$scope.timelineAbsData = data;
	});

	var format = d3.format('f');
	$scope.yAxisTickFormatFunction = function(){
		return function(d) {
			return format(Math.abs(d));
		};
	};

	$scope.$on('barClick', function(event, data) {
		var build = data.point[0];
		updateBreakDown(build);
	});

	$scope.xFunction = function(){
		return function(d) { return d.key; };
	};

	$scope.yFunction = function(){
		return function(d){ return d.value; };
	};

	$scope.offset = 0.025 * screen.width;

	var updateBreakDown = function(build) {
		$scope.build = build;

		$http({method: 'GET', url: '/by_platform', params: {"build": build}})
		.success(function(data) {
			$scope.byPlatform = data;
			$scope.numPlatforms = Object.keys($scope.byPlatform).length;
			$scope.plaformWidth = 0.5 * screen.width * 0.95 / $scope.numPlatforms;

			$http({method: 'GET', url: '/by_priority', params: {"build": build}})
			.success(function(data) {
				$scope.byPriority = data;
				$scope.numPriorities = Object.keys($scope.byPriority).length;
				$scope.priorityWidth = 0.5 * screen.width * 0.95 / $scope.numPriorities;

				$scope.$apply();
			});
		});
	};
}

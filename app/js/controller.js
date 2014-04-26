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
	});

	var format = d3.format('d');
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

		$http({method: 'GET', url: '/platform', params: {"build": build}})
		.success(function(data) {
			$scope.byPlatform = data;
			$scope.numPlatforms = Object.keys($scope.byPlatform).length;
			$scope.plaformWidth = screen.width * 0.95 / $scope.numPlatforms;

			$http({method: 'GET', url: '/priority', params: {"build": build}})
			.success(function(data) {
				$scope.byPriority = data;
				$scope.numPriorities = Object.keys($scope.byPriority).length;
				$scope.priorityWidth = screen.width * 0.95 / $scope.numPriorities;

				$scope.$apply();
			});
		});
	};

	$http.get('/latest').success(function(build) {
		updateBreakDown(build);
	});
}

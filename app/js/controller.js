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
		var latest_build = data[0]["values"][data[0]["values"].length - 1][0];
		updateBreakDown(latest_build);
	});

	$http.get('/abs_timeline').success(function(data) {
		$scope.timelineAbsData = data;
	});

	$scope.relToolTipContentFunction = function() {
		return function(key, build, num) {
			return '<h4>' + num + '% Tests ' + key.replace(', %', '') + '</h4>' +
				'<p>Build ' + build + '</p>';
		};
	};

	$scope.absToolTipContentFunction = function() {
		return function(key, build, num) {
			var failed = $scope.timelineAbsData[1].values;
			var passed = $scope.timelineAbsData[0].values;
			for (var i = 0; i < failed.length; i++) {
				if (passed[i][0] == build) {
					var total;
					if (key === 'Passed') {
						total = -1 * failed[i][1] + parseInt(num, 10);	
					} else {
						total = passed[i][1] + parseInt(num, 10);	
					}					
					return '<h4>' + num + ' of ' + total + ' Tests ' + key + '</h4>' +
						'<p>Build ' + build + '</p>';
				}
			}
		};
	};

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

	var updateBreakDown = function(build) {
		$scope.build = build;

		$http({method: 'GET', url: '/by_platform', params: {"build": build}})
		.success(function(data) {
			$scope.byPlatform = data;
			$scope.numPlatforms = Object.keys($scope.byPlatform).length;
			$scope.plaformWidth = screen.width * 0.95 / 3 /$scope.numPlatforms;

			$http({method: 'GET', url: '/by_priority', params: {"build": build}})
			.success(function(data) {
				$scope.byPriority = data;
				$scope.numPriorities = Object.keys($scope.byPriority).length;
				$scope.priorityWidth = screen.width * 0.95 / 3 / $scope.numPriorities;

				$http({method: 'GET', url: '/by_category', params: {"build": build}})
				.success(function(data) {
					$scope.byCategory = data;
					$scope.numCategories = Object.keys($scope.byCategory).length;
					$scope.categoryWidth = screen.width * 0.95 / 3 / $scope.numCategories;

					$scope.$apply();
				});

			});
		});
	};

	$scope.breakdownToolTipContentFunction = function() {
		return function(status, num) {
			return '<h4>' + parseInt(num, 10) + ' Tests ' + status + '</h4>' +
				'<p>Build ' + $scope.build + '</p>';
		};
	};
}

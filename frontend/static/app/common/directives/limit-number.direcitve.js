(function () {
	"use strict";

	app
		.directive('limitNumber', limitNumber);

	function limitNumber() {
		var directive = {
			restrict: 'A',
			require: '^ngModel',
			scope: {
				numberMax: '@',
				numberMin: '@'
			},
			link: link
		};
		return directive;

		function link(scope, elem, attrs, ctrl) {
			
			scope.$parent.$watch(attrs.ngModel, function(newValue) {
				if (newValue === '' || angular.isUndefined(newValue)) {
					return false;
				}
				vaild(newValue);
			});
			function vaild(newValue) {
				var maxVal = parseInt(scope.numberMax, 10) || 0;
				var minVal = parseInt(scope.numberMin, 10) || 0;
				if (!angular.isUndefined(scope.numberMin) && !angular.isUndefined(scope.numberMax) && minVal > maxVal) {
					return false;
				}
				if (!angular.isUndefined(scope.numberMax)) {
					if (isNaN(newValue)) {
						ctrl.$setViewValue(maxVal);
						ctrl.$render();
					}

					if (newValue > maxVal) {
						ctrl.$setViewValue(maxVal);
						ctrl.$render();
					} 
				}

				if (!angular.isUndefined(scope.numberMin)) {
					if (isNaN(newValue)) {
						ctrl.$setViewValue(minVal);
						ctrl.$render();
					}
					if (newValue < minVal) {
						ctrl.$setViewValue(minVal);
						ctrl.$render();
					} 
				}
			}
			elem.on('blur', function() {
				vaild(this.value);
			});

		}
	}
})();
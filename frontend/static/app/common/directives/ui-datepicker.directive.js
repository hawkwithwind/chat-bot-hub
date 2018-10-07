(function () {
	"user strict";
	angular
		.module('ui.datepicker', [])
		.directive('datePicker', datepicker);

		datepicker.$inject = ['$parse','$rootScope'];
		function datepicker($parse,$rootScope) {
			var directive = {
				restrict: 'A',
				scope: {
					minDate: '=',
					maxDate: '=',
					ngModel: '=',
					dpChange: '&',
					minView: '@',
					format: '@'
				},
				link: link
			};
			return directive;

			function link(scope, elem, attrs) {
				var options = {
					locale: 'zh-cn',
	                format: 'YYYY-MM-DD',
	                showClear: true
				};

				if (scope.format) {
					options.format = scope.format;
				}
				if (scope.minView) {
					options.viewMode = scope.minView;
				}
                // for (var i = 0, l = dpOptions.length; i < l; i++) {
                //     var opt = dpOptions[i];
                //     if (attrs[opt] !== undefined) {
                //         options[opt] = $parse(attrs[opt])();
                //     }
                // }

				elem
					.datetimepicker(options)
	            	.on("dp.change", function (e) {
						if ($rootScope.$$phase) {
							scope.ngModel = e.date ? e.date.format(options.format) : null;
						} else {
							scope.$apply(function() {
								scope.ngModel = e.date ? e.date.format(options.format) : null;
							});
						}
		            });
		        scope.$watch('ngModel', function(date) {
		        	if (date === undefined) {
		        		date = null;
		        	}
		        	elem.data("DateTimePicker").date(date);
		        });
	        	scope.$watch('minDate', function(date) {
	        		if (!date) {
	        			elem.data("DateTimePicker").minDate(false);
	        			return;
	        		}
	        		if (typeof date === 'string') {
	        			date = moment(date);
	        		}
	        		isValid(date) && elem.data("DateTimePicker").minDate(date);
	        	});

	        	scope.$watch('maxDate', function(date) {
	        		if (!date) {
	        			elem.data("DateTimePicker").maxDate(false);
	        			return;
	        		}
	        		if (typeof date === 'string') {
	        			date = moment(date);
	        		}
	        		isValid(date) && elem.data("DateTimePicker").maxDate(date);
	        	});
	        	function isValid(date) {
	        		return angular.isObject(date);
	        	}
			}
		}
})();
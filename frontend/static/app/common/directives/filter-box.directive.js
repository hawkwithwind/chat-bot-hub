(function() {
	"use strict";
	app
		.directive('searchBox', searchBox)
		.directive('showBox', showBox)
		.directive('moreBox', moreBox)

		function searchBox() {
			var directive = {
				restrict: 'EA',
				replace: true,
				transclude: true,
				template: '<div class="search-box table-toolbar form-horizontal" ng-transclude></div>',
				controller: searchBoxCtrl,
				link: link
			}
			searchBoxCtrl.$inject = ['$scope'];
			
			return directive;


			function searchBoxCtrl(scope) {
				scope.itemData = {}

				this.addItem = function(key, val) {
					scope.itemData[key] = val;
				}
				this.getItem = function() {
					return scope.itemData;
				}
			}
			function link(scope, elem, attrs) {
				scope.$watch(attrs.showMore, function(val) {
					if (val) {
						elem.addClass('show-more');
						//$('body').append('<div id="search-box-backdrop" class="fade modal-backdrop in" style="z-index: 2040;"></div>');
					} else {
						elem.removeClass('show-more');
						//$("#search-box-backdrop").remove();
					}
				})				
			}
		}

		function showBox() {
			var directive = {
				restrict: 'EA',
				replace: true,
				transclude: true,
				template: '<div class="show-box" ng-transclude></div>'
			}

			return directive;
		}

		moreBox.$inject = ['$document'];
		function moreBox($document) {
			var directive = {
				require: '^searchBox',
				restrict: 'EA',
				replace: true,
				transclude: true,
				template: '<div class="more-box" ng-transclude></div>',
				link: link
			}

			return directive;

			function link(scope, elem, attrs, ctrl) {
				ctrl.addItem('moreBox', elem);
			}


		}

})();
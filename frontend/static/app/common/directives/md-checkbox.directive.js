(function() {
	"use strict";
	app.directive('mdCheckbox', mdCheckbox);

	function mdCheckbox() {
		var directive = {
			restrict: 'E',
			replace: true,
			scope: {
				ngModel: '=',
				ngText: '='
			},
			template: function(elem, attr) {
				var arr = [
                        '<div class="md-checkbox">',
                            '<input type="checkbox" id="checkbox{{:: random}}" ng-model="ngModel" class="md-check">',
                            '<label for="checkbox{{:: random}}">',
                                '<span></span>',
                                '<span class="check"></span>',
                                '<span class="box"></span>{{::ngText}}',
                            '</label>',
                        '</div>'].join('');
                return arr;
			},
			link: link
		};
		function link(scope, elem, attrs) {
			var random = (Math.random() + '').slice(2);
			scope.random = random;
		}
		return directive;
	}
})();
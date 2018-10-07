(function() {
	"use strict";
	app.directive('mdRadio', mdCheckbox);

	function mdCheckbox() {
		var directive = {
			restrict: 'E',
			replace: true,
			scope: {
				ngModel: '=',
				ngText: '=',
				ngValue: '='
			},
			template: function(elem, attr) {
				var arr = [
                        '<div class="md-radio">',
                            '<input type="radio" value="{{:: ngValue}}" name="{{:: name}}" id="{{:: name}}" ng-model="ngModel" class="md-check">',
                            '<label for="{{:: name}}">',
                                '<span></span>',
                                '<span class="check"></span>',
                                '<span class="box"></span>{{::ngText}}',
                            '</label>',
                        '</div>',
                        ].join('');
                return arr;
			},
			link: link
		};
		function link(scope, elem, attrs) {
			var random = (Math.random() + '').slice(2);
			scope.name = "radio_" + random;
		}
		return directive;
	}
})();
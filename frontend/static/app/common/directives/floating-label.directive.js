(function () {
	"use strict";
	app.directive('formMdFloatingLabel', floatingLabel);

	function floatingLabel() {
		return {
			restrict: 'C',
			link: link
		}
		function link(scope, elem) {
			var $control = elem.find('.form-control');
			$control[0] && $control.val().length > 0 && $control.addClass("edited");
		}
	}
})();
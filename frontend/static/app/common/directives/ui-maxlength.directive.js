(function (argument) {
	angular
		.module('ui.maxlength', [])
		.directive('maxlength', maxlength);

		function maxlength() {
			var directive = {
				restrice: 'A',
				link: link
			}
			return directive;

			function link(scope, elem, attr) {
				var options = {},
					threshold = parseInt(attr['threshold']);
				if (!isNaN(threshold)) {
					options.threshold = threshold;
				}
				elem.maxlength(options);
			}
		}
})();
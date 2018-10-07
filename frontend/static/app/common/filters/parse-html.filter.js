(function() {
	"use strict";
	app.filter('parse', parseHtml);

	parseHtml.$inject = ['$sce'];
	function parseHtml($sce) {
		return function (text) {
		    return $sce.trustAsHtml(text);
		};
	}
})();
(function () {
	"user strict";
	angular
		.module('ui.tree', [])
		.directive('treeList', treeList);

	treeList.$inject = ['$parse'];
	function treeList($parse) {
		var directive = {
			restrict: 'EA',
			scope: {
				data: "=",
				change:'&getChange',
				key:"=key",
				plugins:"@plugins"
			},
			link: link
		};
		return directive;

		function link(scope, elem, attrs) {
			var data = scope.data;
			var change = scope.change;
			if(scope.plugins){
				scope.plugins = scope.plugins.split(',')
			}
			var plugins = scope.plugins||[];
			// 初始化
			elem.jstree({
				"core": {
					"data": data,
					"themes" : {
						"responsive" : false
					}
				},
				"plugins": plugins
			});
			elem.on('changed.jstree', function(e, data) {
				if(data.node){
					change({node:data.node});
				}
			});
			scope.$watch('data',function(newValue,oldValue){
				elem.jstree(true).settings.core.data = newValue;
				elem.jstree(true).refresh();
			})
		}
	}
})();
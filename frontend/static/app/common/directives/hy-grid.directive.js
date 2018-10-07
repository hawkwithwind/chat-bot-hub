(function() {
	"use strict";
	app
		.directive('hGrid', hGrid);
    function hGrid() {
        var directive = {
            restrict: 'E',
            replace: true,
            template: function(elem, attrs) {
                var uiGrid = attrs.gridOptions || 'data.gridOptions';
                if (!angular.isUndefined(attrs.notPage)) {
	                return '<div class="grid-content-box"><div ag-grid="' + uiGrid + '" style="height: 100%;" class="ag-fresh"></div></div>';
	            } else {
	            	var pageSize       = uiGrid + '.paginationPageSize',
	            		curPage        = uiGrid + '.paginationCurPage',
	            		pageSizeList   = uiGrid + '.paginationPageSizes',
	            		totalNum       = uiGrid + '.totalItems',
	            		prefix         = '',
	            		pageChangeFunc = uiGrid + '.interceptorsPagination(' + curPage + ',' + pageSize + ')',
	            		changePageSize = uiGrid + '.interceptorsPagination(1,' + pageSize + ', true)';
	            	return '<div class="grid-content-box">' + 
	            			'<div ag-grid="' + uiGrid + '" style="height: 100%;" class="ag-fresh"></div>' + 
	            			'<div class="pagination-box">' + 
	            				'<uib-pagination force-ellipses="true" items-per-page="' + pageSize + '" ng-change="' + pageChangeFunc + '" total-items="' + totalNum + '" ng-model="' + curPage + '" max-size="5" class="pagination-sm" boundary-links="true" force-ellipses="true"></uib-pagination>' + 
	            				'<select ng-change="' + changePageSize + '" ng-model="' + pageSize + '" ng-options="item for item in ' + pageSizeList + '"></select/>' + 
	            				'<span style="vertical-align:9px; padding-left: 20px;">总计: <span ng-bind="' + totalNum + ' || 0"></span></span>' + 
	            			'</div>' +
	            			'</div>';
	            }
            },
            link: function($scope, elem, attrs) {
            	var uiGrid = attrs.gridOptions || 'data.gridOptions',
            		gridOptions = $scope.$eval(uiGrid);
            	if (angular.isObject(gridOptions)) {
                    if (angular.isFunction(gridOptions.onPaginationChanged)) {
                		gridOptions.interceptorsPagination = function(curPage, pageSize) {
                			gridOptions.paginationCurPage = curPage;
                			gridOptions.onPaginationChanged(curPage, pageSize);
                		};
                    }
                    gridOptions.$element = elem;
            	}
            	
            }
        };
        return directive;
    }
})();
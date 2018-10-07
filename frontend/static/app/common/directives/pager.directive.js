(function() {
	"use strict";
	app.directive('tablePager', function(){
        return{
            restrict: "E",
            replace: true,
            scope: {
            	pageChange: '&?',
            	toggle: '&?', //分页显示隐藏时调用
            	total : '=?',
            	ngModel: '=?'
            },
            template: '<ul class="pagination"></ul>',
            link: function(scope, elem, attrs){
                var num = parseInt(attrs['show']) || 2;
                if (scope.ngModel && isNaN(scope.ngModel)) {
                	scope.ngModel = parseInt(scope.ngModel)
            	} else {
            		scope.ngModel = 1;
            	}

                if (!angular.isFunction(scope.toggle)) {
                	scope.toggle = function() {};
                }
                function bulidPager(curPage, total, num) {
                	 if(scope.total <= 1){
	                    elem.hide();
	                    scope.toggle({'status': false});
	                    return false;
	                } else {
	                	elem.show();
	                	scope.toggle({'status': true});
	                }
	                var prevPage = true ,
                    	nextPage = true,
                	 	html = '',
            	 	 	page = 0 ;
            	 	 total = parseInt(total);
	                if(curPage <= 1){
	                    curPage = 1;
	                    prevPage = false;
	                }
	                if(curPage >= total){
	                    curPage = total;
	                    nextPage = false;
	                }
	                for(var i = num; i >= 1 ; i--){
	                    page = curPage - i;
	                    if(page < 1){
	                        continue;
	                    }
	                    else{
	                        html += '<li title="第' + page + '页"><a data-page="' + page + '" href="javascript:;">' + page + '</a></li>';
	                    }
	                }
	                html += '<li class="active"><a href="javascript:;">' + curPage + '</a></li>';
	                for(i = 1; i <= num; i++){
	                    page = i + curPage;
	                    if(page > total){
	                        break;
	                    }
	                    html += '<li title="第' + page + '页"><a data-page="' + page + '" href="javascript:;">' + page + '</a></li>';
	                }

	                if(prevPage){
						html = '<li class="first" title="上一页"><a data-page="' + (curPage - 1) + '" href="javascript:;"><i class="fa fa-angle-left"></i></a></li>' + html
	                }else{
	                    html = '<li class="first disabled" title="上一页"><a href="javascript:;"><i class="fa fa-angle-left"></i></a></li>' + html;
	                }

	                if(curPage == 1){
	                    html = '<li class="prev disabled" title="首页"><a href="javascript:;">« 首页</a></li>' + html;
	                }else{
	                    html = '<li class="prev" title="首页"><a data-page="1" href="javascript:;">« 首页</a></li>' + html;
	                }

	                if(nextPage){
						html = html + '<li class="next"  title="下一页"><a data-page="' + (curPage + 1) + '" href="javascript:;"><i class="fa fa-angle-right"></i></a></li>'
	                }else{
	                    html = html + '<li class="next disabled" title="下一页"><a href="javascript:;"><i class="fa fa-angle-right"></i></a></li>';
	                }

	                if(curPage == total){
	                    html = html + '<li class="last disabled" title="尾页"><a href="javascript:;">尾页 »</a></li>';
	                }else{
	                    html = html + '<li class="last" title="尾页"><a data-page="' + total + '" href="javascript:;">尾页 »</a></li>';
	                }
	                elem.html(html);
            	}
            	scope.$watch('total + ngModel', function(v) {
            		bulidPager(scope.ngModel, scope.total, num);
            	})
            	/*scope.$watch('ngModel', function(v) {
            		bulidPager(v, scope.total, num);
            	})*/
                elem.on('click', "a[data-page]", function() {
                	var $this = $(this), pageIndex = $this.attr('data-page');
                	pageIndex = isNaN(pageIndex) ? 1 : parseInt(pageIndex);
                	if (angular.isFunction(scope.pageChange)) {
                		scope.pageChange({'page': pageIndex});
                	}
                	scope.ngModel = pageIndex;
                	scope.$apply();
                })
            }
        }
    })


})();
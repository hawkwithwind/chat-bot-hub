(function () {
	"use strict";

	angular
		.module('viewLayout', [])
		.directive('hyFullheight', hyFullheight)
		.directive('hyView', hyView)
		.controller('hyViewCtrl', hyViewCtrl)
		.directive('hyViewHeader', hyViewHeader)
		.directive('hyViewFooter', hyViewFooter)
		.directive('hyViewContent', hyViewContent)
		.directive('hyViewDetailGrid', hyViewDetailGrid);

    hyFullheight.$inject = ['$window', '$timeout'];
    function hyFullheight ($window, $timeout) {
        var directive = {
            restrict: 'AE',
            scope: {
                hyFullheightIf: '&'
            },
            link: link
        };
        return directive;

        function link (scope, elem, attrs) {
            var $win = $($window);
            var $document = $(document);
            var exclusionItems;
            var exclusionHeight;
            var diffHeight;
            var timeoutId;
            var setHeight = true;
            var page;
            scope.initializeWindowSize = function () {
            	$timeout.cancel(timeoutId);
                timeoutId = $timeout(function () {
                	diffHeight = parseInt(attrs.hyDiffHeight) || 0;
                    exclusionHeight = 0;
                    // 切换auto和fullheight
                    if (attrs.hyFullheightIf) {
                        scope.$watch(scope.hyFullheightIf, function (newVal, oldVal) {
                            if (newVal && !oldVal) {
                                setHeight = true;
                            } else if (!newVal) {
                                $(elem).css('height', 'auto');
                                setHeight = false;
                            }
                        });
                    }
                    // 排除[元素]的高度
                    if (attrs.hyFullheightExclusion) {
                        var exclusionItems = attrs.hyFullheightExclusion.split(',');
                        angular.forEach(exclusionItems, function (_element) {
                            // outerHeight contentWidth + p + b + m
                            exclusionHeight = exclusionHeight + $(_element).outerHeight(true);
                        });
                    }
                    if (attrs.hyFullheight == 'window') {
                        page = $win;
                    } else {
                        page = $document;
                    }

                    scope.$watch(function () {
                        scope.__height = page.height();
                    });
                    if (setHeight) {
                        // 设置为auto方便获取高度
                        $(elem).css('height', 'auto');
                        // doucumnet的高度小于window的高度 取window的高度
                        if (page.innerHeight() < $win.innerHeight()) {
                            page = $win;
                        }
                        $(elem).css('height', page.innerHeight() - exclusionHeight - diffHeight);
                    }
                }, 300);
            };

            scope.initializeWindowSize();
            scope.$watch('__height', function (newHeight, oldHeight) {
                scope.initializeWindowSize();
            });
            $win.on('resize', function () {
                scope.initializeWindowSize();
            });
        }
    }
	function hyView() {
		var directive = {
			restrict: 'EA',
			transclude: true,
			replace: true,
			template: '<div class="hy-view" ng-transclude></div>',
			scope: {},
			controller: hyViewCtrl,
			link: link
		};
		return directive;

		function link(scope, elem, attrs) {
			elem.css({'height': '100%', 'overflow': 'hidden'}).parent().css('height', '100%');
			var $content = elem.find('.hy-view-content');
		}
	}
	hyViewCtrl.$inject = ['$scope'];
	function hyViewCtrl($scope) {
		$scope.headerHeight = 0;
		$scope.footerHeight = 0;
		var contentScope = {};
		this.setHeaderHeight = function(height) {
			$scope.headerHeight = height;
		};
		this.setFooterHeight = function(height) {
			$scope.footerHeight = height;
		};
		this.setContent = function(cScope) {
			contentScope = cScope;
		};
		this.setContentHide = function(isHide) {
			contentScope.isHide = isHide;
		};
		this.getHeaderHeight = function() {
			return $scope.headerHeight;
		};
		this.getFooterHeight = function() {
			return $scope.footerHeight;
		};

		$scope.$watch('headerHeight + footerHeight', function() {
			contentScope.styleObj = {
				'marginTop': $scope.headerHeight * -1, 
				'paddingTop': $scope.headerHeight,
				'marginBottom': $scope.footerHeight * -1,
				'paddingBottom': $scope.footerHeight
			};
		});
	}
	hyViewHeader.$inject = ['$timeout'];
	function hyViewHeader($timeout) {
		var directive = {
			restrict: 'EA',
			transclude: true,
			replace: true,
			require: '^hyView',
			template: '<div class="hy-view-header" ng-transclude></div>',
			scope:{
				hyViewHeight: '@',
				hyDiffHeight: '@'
			},
			link: link
		};
		return directive;

		function link(scope, elem, attrs, ctrl) {
			// 延时计算高度
			var outerHeight = 0,
				diffHeight = parseInt(scope.hyDiffHeight) || 0,
				headerHeight = scope.hyViewHeight && parseInt(attrs.hyViewHeight);

			scope.$watch('hyViewHeight', calcHeight);

			function calcHeight(headerHeight) {
				headerHeight = parseInt(headerHeight);
					if (headerHeight) {
						outerHeight = headerHeight;
					} else {
						outerHeight = $(elem).outerHeight(true);
					}
					ctrl.setHeaderHeight(outerHeight + diffHeight);
				$timeout(function() {
				});
			}
		}
	}

	hyViewFooter.$inject = ['$timeout'];
	function hyViewFooter($timeout) {
		var directive = {
			restrict: 'EA',
			transclude: true,
			replace: true,
			require: '^hyView',
			scope: {
				hyDiffHeight: '@',
				hyViewHeight: '@'
			},
			template: '<div class="hy-view-footer" ng-style="styleData" ng-transclude></div>',  // hy-view-height
			link: link
		};
		return directive;

		function link(scope, elem, attrs, ctrl) {
			// 延时计算高度
			var outerHeight = 0,
				diffHeight = parseInt(scope.hyDiffHeight) || 0,
				footerHeight = attrs.hyViewHeight && parseInt(scope.hyViewHeight);

			scope.$watch('hyViewHeight', calcHeight);

			function calcHeight(footerHeight) {
				footerHeight = parseInt(footerHeight);
					if (footerHeight) {
						outerHeight = footerHeight;
					} else {
						outerHeight = $(elem).outerHeight(true);
					}
					ctrl.setFooterHeight(outerHeight + diffHeight);
					scope.styleData = { height: outerHeight + diffHeight};
				$timeout(function() {
				});
			}
		}
	}
	function hyViewContent() {
		var directive = {
			restrict: 'EA',
			transclude: true,
			replace: true,
			scope: {},
			require: '^hyView',
			template: '<div ng-style="styleObj" ng-hide="isHide" class="hy-view-content" ng-transclude></div>',
			link: link
		};
		return directive;

		function link(scope, elem, attrs, ctrl) {
			elem.css({height: '100%'});
			ctrl.setContent(scope);
		}
	}

	hyViewDetailGrid.$inject = ['$timeout'];
	function hyViewDetailGrid($timeout) {
		var directive = {
			restrict: 'EA',
			require: '^hyView',
			replace: true,
			transclude: true,
			scope: {
				hyDiffHeight: '@'
			},
			template: function(elem, attrs) {
				var splitbar = '<div class="splitbar"> ' +
									'<a ng-click="upArrow($event);"><span class="glyphicon glyphicon-chevron-up"></span></a>' +
									'<a ng-click="downArrow($event);"><span class="glyphicon glyphicon-chevron-down"></span></a>' +
								'</div>';
				return 	'<div class="hy-view-detail-grid">'+
							splitbar + 
							'<hy-view-footer hy-diff-height="{{hyDiffHeight}}" hy-view-height="{{hyViewDetailGrid.footerHeight}}">' +
								'<div style="height: 100%; padding-top: 10px;" ng-hide="!hyViewDetailGrid.expand" ng-transclude></div>' +
							'</hy-view-footer>' + 
						'</div>';
			},
			link: link
		};
		return directive;
		function link(scope, elem, attrs, ctrl) {
			var height = calcHeight(),
				tmpTop = height,
				isHideContent = false;

			scope.hyViewDetailGrid = {
				footerHeight: height,
				expand: true,
				changeFooterHeight: function() {
					scope.hyViewDetailGrid.expand = !scope.hyViewDetailGrid.expand;
					scope.hyViewDetailGrid.footerHeight = scope.hyViewDetailGrid.expand ? height: 34;
				}
			};
			scope.downArrow = function(e) {
				if (isHideContent) {
					ctrl.setContentHide(false);
					scope.hyViewDetailGrid.footerHeight = tmpTop;
					isHideContent = false;
				} else {
					scope.hyViewDetailGrid.expand = false;
					scope.hyViewDetailGrid.footerHeight = 12;
				}
				e.stopPropagation();
			};
			scope.upArrow = function(e) {
				// 如果是收起状态
				if (!scope.hyViewDetailGrid.expand) {
					scope.hyViewDetailGrid.expand = true;
					scope.hyViewDetailGrid.footerHeight = tmpTop;
				} else {
					ctrl.setContentHide(true);
					isHideContent = true;
					scope.hyViewDetailGrid.footerHeight = $("#container").height() - ctrl.getHeaderHeight();
				}
				e.stopPropagation();
			};

			splitbarFunc();
			
			function splitbarFunc() {
				var params = {
					top: 0,
					left: 0,
					minHeight: 150,
					maxHeight: 0
				};
				//$timeout(function() {
					params.maxHeight = ctrl.getHeaderHeight() + 100 + 56;
				//}, 100);
				var $splitbar = elem.find(".splitbar");

				$splitbar.on('mousedown', function(e) {
					// 如果是收起 不再可以拖动
					if (!scope.hyViewDetailGrid.expand || isHideContent) {
						return false;
					}
					var $this = $(this),
						// 屏幕高度
						windowHeight = $(window).height(),
						// containter高度， 去除header后
						containterHeight = $("#container").height();

					params.top = this.offsetTop;
					//params.currentX = e.clientX;
	        		params.currentY = e.clientY;
	        		$splitbar.css('position', 'absolute');
					$(window).on('mousemove', function(e) {
						//var nowX = e.clientX,
			            var nowY = e.clientY;
			            //var disX = nowX - params.currentX,
			            var disY = nowY - params.currentY;
			            var parseTop = parseInt(params.top);
			            var topValue = parseTop + disY;
			            // 限制最小100px  屏幕高度 - 鼠标Y轴坐标 
			            if (windowHeight - nowY < params.minHeight) {
			            	topValue = (containterHeight - params.minHeight);
			            }
			            // 限制拖动最大高度 view-header的高度 + grid header高度 + 行高 + 分页高度
			            if ( params.maxHeight + windowHeight - containterHeight > nowY ) {
			            	topValue = params.maxHeight;
			            }
						$this.css({
							top: topValue
						});
						tmpTop = containterHeight - topValue;
						return false;
					}).on('mouseup', function() {
						$(window).off('mousemove');
						$(window).off('mouseup');

						scope.$apply(function() {
							scope.hyViewDetailGrid.footerHeight = tmpTop;
						});

						// 为了自适应高度，去除绝对定位的值
						$splitbar.removeAttr('style');
					});
				});
			}
			function calcHeight() {
				if (attrs.height.indexOf('%') > -1) {
					return $('#container').innerHeight() * (parseInt(attrs.height) / 100);
				} else {
					return parseInt(attrs.height) || 300;
				}
			}
		}
	}
})();
angular.module("app")
	.directive("hySidebar", hySidebar);
	hySidebar.$inject = ["APP_MEDIAQUERY", "requestAnimationFrame", '$rootScope'];
	function hySidebar (APP_MEDIAQUERY, requestAnimationFrame, $rootScope) {
		var directive = {
				restrict: 'EA',
				link: link
			};

		return directive;

		function link(scope, $elem, attr) {
			var $body = $('body'), 
				$sildeMenu = $(".page-sidebar-menu"),
				$pageSidebarWrapper = $(".page-sidebar-wrapper"),
				breakpoint = APP_MEDIAQUERY.desktop,
				isNavToogle = false;
			// 左侧导航
			$elem.on("click", "li>.nav-toggle, li > a > span.nav-toggle", sildeExpand);
			// 导航 toggle 按钮
			$body.on("click", ".sidebar-toggler", sildeToggle);

			$rootScope.$on("$locationChangeSuccess", function() {
                var hash =  window.location.hash;
				$elem.find("a").each(function(b) {
					var $this = $(this),
						$subMenu = null;
                    if (this.hash === hash ) {
                    	if (!$(this).closest('.sub-menu').is(":visible")) {
                    		sildeExpand.bind($this.closest(".sub-menu").prev())();
                    	}
                    }
                    $subMenu = null;
                });
            });

			function bindSildeHover() {
				$pageSidebarWrapper.on('mouseenter', sildeEnter).on('mouseleave', sildeLeave);
			}
			function unBindSildeHover() {
				$pageSidebarWrapper.off('mouseenter').off('mouseleave');
			}
 			function sildeEnter(e) {
				var $this = $(this);
				if ($sildeMenu.hasClass("page-sidebar-menu-closed")) {
					$sildeMenu.removeClass("page-sidebar-menu-closed");	
				}
			}
			function sildeLeave(e) {
	            $sildeMenu.addClass("page-sidebar-menu-closed");
			}
			function sildeExpand(e) {
				isNavToogle = true;
	            var $navLink = $(this).closest(".nav-item").children(".nav-link"),
	            	viewPortWidth = 1280;
            	var hasSubMenu = $navLink.next().hasClass("sub-menu");

                if (hasSubMenu === false) {
                	return viewPortWidth < breakpoint && $(".page-sidebar").hasClass("in") && $(".page-header .responsive-toggler").click();
                }

                var $parentSubmenu = $navLink.parent().parent(),
                    $childrenSubmenu = $navLink.next(),
                    slideSpeed = 200,
                    isKeepExpanded = true;  //是否可以同时打开多个

                if (isKeepExpanded) {
                	$parentSubmenu.children("li.open").children(".sub-menu:not(.always-open)").slideUp(slideSpeed);
                	$parentSubmenu.children("li.open").removeClass("open");
                }
                if ($childrenSubmenu.is(":visible")) {
                	$navLink.parent().removeClass("open");
                	$childrenSubmenu.slideUp(slideSpeed);
                } else {
                	if (hasSubMenu) {
                		$navLink.parent().addClass("open");
                		$childrenSubmenu.slideDown(slideSpeed);
                	}
                }
                e && e.preventDefault();
			}
			function sildeToggle(e) {
	            if ($body.hasClass("page-sidebar-closed")) {
	            	$body.removeClass("page-sidebar-closed");
	            	$sildeMenu.removeClass("page-sidebar-menu-closed");
	            	unBindSildeHover();
	            } else {
					$body.addClass("page-sidebar-closed");
	            	$sildeMenu.addClass("page-sidebar-menu-closed");
	            	bindSildeHover();
	            }
			}
			$pageSidebarWrapper.children().css({
				'transform': 'translateX(0px)'
			});
			requestAnimationFrame(function() {
		        $elem.children().each(function(index) {
		        	$(this).css({
		        		'transform': 'translateY(0px)',
			        	'opacity': 1,
			        	'transition-delay': 0.5 + index * .1 + 's'
			        });
		        });
			});
		}
	}
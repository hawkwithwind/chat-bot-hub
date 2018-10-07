(function () {
	angular.module('ui.ripple', [])
		.directive('ripple', ripple);
		function ripple() {
			var directive = {
				restrict: 'EA',
				link: link
			}
			return directive;

			function link(scope, elem, attrs) {
				var animationLibrary = 'animate';
		        $.easing.easeOutQuart = function (x, t, b, c, d) {
		            return -c * ((t = t / d - 1) * t * t * t - 1) + b;
		        };
		        if (elem.is(':not([disabled],.disabled)')) {
		        	elem.on('mousedown', function (e) {
			            var button = $(this);
			            var touch = $('<touch><touch/>');
			            var size = button.outerWidth() * 1.8;
			            var complete = false;
			            $(document).on('mouseup', function () {
			                var a = { 'opacity': '0' };
			                if (complete === true) {
			                    size = size * 1.33;
			                    $.extend(a, {
			                        'height': size + 'px',
			                        'width': size + 'px',
			                        'margin-top': -size / 2 + 'px',
			                        'margin-left': -size / 2 + 'px'
			                    });
			                }
			                touch[animationLibrary](a, {
			                    duration: 500,
			                    complete: function () {
			                        touch.remove();
			                    },
			                    easing: 'swing'
			                });
			            });
			            touch.addClass('touch').css({
			                'position': 'absolute',
			                'top': e.pageY - button.offset().top + 'px',
			                'left': e.pageX - button.offset().left + 'px',
			                'width': '0',
			                'height': '0'
			            });
			            button.get(0).appendChild(touch.get(0));
			            touch[animationLibrary]({
			                'height': size + 'px',
			                'width': size + 'px',
			                'margin-top': -size / 2 + 'px',
			                'margin-left': -size / 2 + 'px'
			            }, {
			                queue: false,
			                duration: 500,
			                'easing': 'easeOutQuart',
			                'complete': function () {
			                    complete = true;
			                }
			            });
			        });
		        }
			}
		}
})()
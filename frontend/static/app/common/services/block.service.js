(function () {
	'use strict';
	angular.module('ui.block', [])
		.factory('block', block);

		function block() {
			var targetList = {
				'window': function () {
					return $(window);
				},
				'container': function() {
					return $("#container");
				}
			};
			var factory = {
				quickBlock: function(key, options) {
					options = angular.extend({}, options);
					options.target = targetList[key]();
					factory.blockUI(options);
				},
				blockUI: function(options) {
		            options = $.extend({}, options);
		            var message = '<div class="loading-message ' + (options.boxed ? "loading-message-boxed" : "") + '">' +
					            		'<div class="block-spinner-bar">' +
					            		'<div class="bounce1"></div>' +
					            		'<div class="bounce2"></div>' +
					            		'<div class="bounce3"></div>' +
					            		'</div>' +
					            	'</div>';
		            if (options.target) {
		                var $target = $(options.target);
		                if ($target.height() <= $(window).height()) {
		                	options.cenrerY = true;
		                }
		                $target.block({
		                    message: options.message || message,
		                    baseZ: options.zIndex ? options.zIndex : 10000,
		                    centerY: options.cenrerY ? options.cenrerY : false,
		                    css: {
		                        top: "10%",
		                        border: "0",
		                        padding: "0",
		                        backgroundColor: "none"
		                    },
		                    overlayCSS: {
		                        backgroundColor: options.overlayColor ? options.overlayColor : "#555",
		                        opacity: options.boxed ? .05 : .1,
		                        cursor: "wait"
		                    }
		                });
		            } else {
		            	$.blockUI({
			                message: message,
			                baseZ: options.zIndex ? options.zIndex : 10000,
			                css: {
			                    border: "0",
			                    padding: "0",
			                    backgroundColor: "none"
			                },
			                overlayCSS: {
			                    backgroundColor: options.overlayColor ? options.overlayColor : "#555",
			                    opacity: toptionsboxed ? .05 : .1,
			                    cursor: "wait"
			                }
			            });
		            }
		        },
		        unblockUI: function(target) {
		            if (target) {
			            $(target).unblock();
		            } else {
		            	$.unblockUI();
		            }
		        }
			};
			return factory;
		}
})()
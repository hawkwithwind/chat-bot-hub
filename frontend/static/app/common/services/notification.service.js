(function (window) {
	"use strict";
	angular.module('ui.notification', [])
		.factory('notification', notification);

		notification.$inject = ['$rootScope', '$q'];

		function notification($rootScope, $q) {
			bootbox.setLocale("zh_CN");
			var result = {
				'alert': function(title) {
					title = '<div style="word-break: break-all; margin-top: 12px">' + title + '</div>';
					var defered = $q.defer();
					$rootScope.$evalAsync(function(){
						bootbox.alert(title, function() {
							defered.resolve();
						});
					});
					return defered.promise;
				},
				'confirm': function(title , custom) {
					var defered = $q.defer();
					$rootScope.$evalAsync(function(){
						bootbox.confirm(title, function(result) {
							if (result) {
								defered.resolve();
							} else {
								defered.reject();
							}
						});
					});
					return defered.promise;
				},
				'prompt': function(title, value, required) {
					var defered = $q.defer();
					bootbox.prompt({
						title: title,
						value: value,
						callback: function(result) {
							// 取消
						    if (result === null) {
								defered.reject();
						    } else {
						    	if (angular.isFunction(required)) {
						    		if (required(result) === false) {
						    			return false;
						    		}
						    	} else if (required) {
						    		return false;
						    	}
						    }
						    defered.resolve(result);
						}
					});
					return defered.promise;
				},
				'promptArea': function(title, val, obj) {
					var defined = $q.defer();
					obj = obj || {};
					var html = '<textarea id="bootbox-prompt" {maxlength} {placeholder} rows="5" class="bootbox-input bootbox-input-text form-control">'+ (val || '') + '</textarea>';
					for(var item in obj) {
						if (obj[item]) {
							html = html.replace('{' + item + '}', item + '="' + obj[item] + '"');
						}
					}
					bootbox.dialog({
					  	message: html,
						title: title,
						buttons: {
							danger: {
								label: "取消",
								className: "btn-default",
								callback: function () {
									defined.reject();
								}
							},
							success: {
								label: "确认",
								className: "btn-primary",
								callback: function (e) {
									var val = $('#bootbox-prompt').val();
									val = val.trim();
									if (angular.isFunction(obj.isClose)) {
										if (obj.isClose(val) === false) {
											return false;
										} else {
											defined.resolve(val);
										}
									} else {
										defined.resolve(val);
									}
								}
							}
						}
					});
					return defined.promise;
				},
				'custom': function(options) {
					bootbox.dialog(options);
				}
			}
			return result;
		}
})(window);
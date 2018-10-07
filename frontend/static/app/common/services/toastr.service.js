(function () {
	angular.module('ui.toastr', [])
		.factory("toastr", toastr);

		var toastr = window.toastr || {};
		function toastr() {
			setToastr();
			var service = {
				show: function (message, type, title) {
					type = type || 'success'; // success info warning error
					toastr[type] && toastr[type](message, title);
				},
				success: function(message, title, timeout) {
					setTimout(timeout);
					toastr.success(message, title);
				},
				info: function(message, title, timeout) {
					setTimout(timeout);
					toastr.info(message, title);
				},
				warning: function(message, title, timeout) {
					setTimout(timeout);
					toastr.warning(message, title);
				},
				error: function(message, title, timeout) {
					setTimout(timeout);
					toastr.error(message, title);
				},
				reqError: function(data) {
					setTimout();
					toastr.error(data.message);
				}
			}
			return service;

			function setToastr(options) {
				var defaultOptions = {
					"closeButton": true,
					"debug": false,
					"positionClass": "toast-top-center",
					"timeOut": 3000
					/*"onclick": null,
					"showDuration": "1000",
					"hideDuration": "1000",
					"extendedTimeOut": "1000",
					"showEasing": "swing",
					"hideEasing": "linear",
					"showMethod": "fadeIn",
					"hideMethod": "fadeOut"*/
				};
				
				//options = angular.extend(defaultOptions, options);
				toastr.options = defaultOptions;
			}
			function setTimout (timeout) {
				timeout = timeout || 3000;
				toastr.options.timeOut = timeout;
			}
		}
})()
(function () {
	app
		.directive('fileInput', fileInput);

		function fileInput() {
			var directive = {
				restrice: 'A',
				replace: true,
				scope: {
					changeFile: '&'
				},
				link: link
			};
			return directive;

			function link(scope, elem, attr) {
				var $file = $('<input type="file">');
				elem.addClass('file-input').append($file);
				$file.on('change', function(e){
					var files = this.files;
					if (angular.isFunction(scope.changeFile)) {
						// 是否清空file文件 避免第二次上次同名文件是不触发 onchange事件
						if (!scope.changeFile({'files': files})) {
							setTimeout(function() {
								$file.val('');
							}, 1000);
						}
					}
				});
			}
		}
})();
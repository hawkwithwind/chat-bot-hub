(function (argument) {
	app
		.directive('setUsername', setUsername);
	setUsername.$inject = ['$rootScope'];
	function setUsername($rootScope) {
		var directive = {
			restrice: 'A',
			link: link
		};
		return directive;

		function link(scope, elem, attr) {
			//elem.text($rootScope.username);
			//userModel.getUserInfo.query(function(response) {
			//	elem.text(response.data.accountName);
			//});
		}
	}
})();
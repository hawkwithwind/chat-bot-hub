(function () {
  'use strict';
  app.factory('buildModel', buildModel);
  buildModel.$inject = ['$resource', 'URL'];
  function buildModel($resource, URL) {
    return function(key, params, method) {
      var defaultMethod = {
	'query': {method: 'GET', isArray: false}, 
	'update': {method: 'POST'},
	'updateArray':{method: 'POST',isArray:true},
	'queryArray': {method: 'GET', isArray: true},
	'put':{method: 'put'},
	'post': {method: 'POST'},
	'remove':{method: 'delete'},
	'jsonp':{method: 'JSONP'}
      };
      if (!URL[key]) {
	console.error('在url.server中找不到名为: [' + key +']的url');
      }
      return $resource(URL[key], params, angular.extend(defaultMethod, method));
    };
  }
})();

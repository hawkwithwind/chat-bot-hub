(function () {
  'use strict';
  app.factory('buildModelResId', buildModelResId);
  buildModelResId.$inject = ['$resource', 'URL'];
  function buildModelResId($resource, URL) {
    return function(key, rid, params, method) {
      var defaultMethod = {
	'query': {method: 'GET', isArray: false}, 
	'update': {method: 'POST'},
	'updateArray':{method: 'POST',isArray:true},
	'queryArray': {method: 'GET', isArray: true},
	'put':{method: 'put'},
	'remove':{method: 'delete'}
      };
      if (!URL[key]) {
	console.error('在url.server中找不到名为: [' + key +']的url');
      }
      return $resource(URL[key]+'/'+rid, params, angular.extend(defaultMethod, method));
    };    
  }
})();

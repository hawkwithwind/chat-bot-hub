(function () {
  'use strict';
  
  app.factory('buildPromise', buildPromise);
  buildPromise.$inject = ['$q', 'toastr'];
  
  function buildPromise($q, toastr) {
    
    return function(resource, data, key) {
      if (!resource) {
	console.error('找不到指定的resource对象');
	return false;
      }
      var defered = $q.defer();
      var params = angular.extend({}, data);
      key = key || 'query';

      resource[key](params, function (d) {
	defered.resolve(d);
      }, function (d) {
	defered.reject(d);
	//toastr.reqError(d);
      });
      return defered.promise;
    };
  }
})();

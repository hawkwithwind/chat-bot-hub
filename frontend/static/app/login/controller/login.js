(function() {
  "use strict"
  app.controller("loginCtrl", loginCtrl);
  loginCtrl.$inject = ["$scope",  "toastr", "buildModel",
		       "buildPromise", "tools", "buildModelResId", "$sce",
		       "$httpParamSerializer", "$http", "$window", "$timeout"];
  function loginCtrl($scope, toastr,
		     buildModel, buildPromise, tools, buildModelResId, $sce,
		     $httpParamSerializer, $http, $window, $timeout) {
    $scope.body = {};
    
    $scope.login = function(data) {
      var url = 'login';

      $http.post(url, JSON.stringify(data)).
	success(function(data, status, headers, config) {
	  //$scope.bodypretty = $sce.trustAsHtml($scope.pretty(data));
	  $timeout(function() {
	    $window.location.href = '#/app/manage/botslist';
	  }, 100);
	}).
	error(function(data, status, headers, config) {
	  //$scope.bodypretty = $sce.trustAsHtml($scope.pretty(data));
	});
    }

    $scope.githublogin = function(data) {
      $http.get('/githublogin').success(function(data, status, headers, config) {
	console.log(data);
	console.log(status);
	console.log(headers);
	console.log(config);
      })
    }

    $scope.pretty = function (json) {
      if (typeof json != 'string') {
        json = JSON.stringify(json, undefined, 2);
      }
      
      json = json.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
      return json.replace(/("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g, function (match) {
        var cls = 'number';
        if (/^"/.test(match)) {
          if (/:$/.test(match)) {
            cls = 'key';
          } else {
            cls = 'string';
          }
        } else if (/true|false/.test(match)) {
          cls = 'boolean';
        } else if (/null/.test(match)) {
          cls = 'null';
        }
        return '<span class="' + cls + '">' + match + '</span>';
      });
    }    
  }
})();


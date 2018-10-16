(function() {
  "use strict"
  app.controller("botslistCtrl", botslistCtrl);
  botslistCtrl.$inject = ["$scope",  "toastr", "buildModel",
		       "buildPromise", "tools", "buildModelResId", "$sce",
		       "$httpParamSerializer", "$http", "$window"];
  function botslistCtrl($scope, toastr,
			buildModel, buildPromise, tools, buildModelResId, $sce,
			$httpParamSerializer, $http, $window) {
    $scope.body = {};
    
    $scope.tsToString = function(unix_timestamp) {
      var date = new Date(unix_timestamp);

      var year = date.getFullYear();
      var month = "0" + (date.getMonth()+1);
      var datestr = "0" + date.getDate();
      var hours = "0" + date.getHours();
      var minutes = "0" + date.getMinutes();
      var seconds = "0" + date.getSeconds();
      
      return year+'-'+month.substr(-2)+'-'+datestr.substr(-2)+' ' +
	hours.substr(-2)+':'+minutes.substr(-2)+':'+seconds.substr(-2);
    }

    $scope.initView = function(data) {
      $scope.body = data.body.botsInfo;
    }

    buildPromise(buildModel('consts'))
      .then(function(data) {
	$scope.consts = data.body;
	
	buildPromise(buildModel('bots'))
	  .then(function(data) {
	    $scope.initView(data);
	  });
      });
    
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

    $scope.showQQLogin = function (row)  {
      
    }

    $scope.wechatLogin = function (row) {
      buildModel('loginwechat', {clientId: row.clientId}).post(function(data) {
	$scope.bodypretty = $scope.pretty(data);
      });
    }
  }
})();


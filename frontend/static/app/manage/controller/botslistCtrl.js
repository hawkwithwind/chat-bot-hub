(function() {
  "use strict"
  app.controller("botslistCtrl", botslistCtrl);
  botslistCtrl.$inject = ["$scope", "$modal", "toastr", "buildModel",
		       "buildPromise", "tools", "buildModelResId"];
  function botslistCtrl($scope, $modal, toastr,
			buildModel, buildPromise, tools, buildModelResId) {
    $scope.body = {};
    
    $scope.tsToString = function(unix_timestamp) {
      if(unix_timestamp === undefined) { return "" }
      
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
      $scope.body = data.body;
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

    $scope.showQQLogin = function(row)  {
      $modal.open({
	templateUrl: 'loginQQTemplate',
	controller: qqLoginCtrl,
	resolve: {
	  clientId: () => row.clientId
	}
      });
    }

    $scope.showWechatLogin = function(row)  {
      $modal.open({
	templateUrl: 'loginWechatTemplate',
	controller: wechatLoginCtrl,
	resolve: {
	  clientId: () => row.clientId
	}
      });
    }

    $scope.wechatLogin = function(row) {
      buildModel('botlogin', {
	clientId: row.clientId,
	clientType: row.clientType,
	botId: row.botId
      }).post(function(data) {
	log.console(data);
      });
    }

    $scope.botAction = function(row) {
      $modal.open({
	templateUrl: 'botActionTemplate',
	controller: botActionCtrl,
	resolve: {
	  clientId: () => row.clientId,
	  login: () => row.login
	}
      });
    }
  }

  app.controller("qqLoginCtrl", qqLoginCtrl);
  qqLoginCtrl.$inject = ["$scope", "$uibModalInstance", "toastr", "buildModel", "buildPromise", "tools", "clientId", "botId"];
  function qqLoginCtrl($scope, $uibModalInstance, toastr, buildModel, buildPromise, tools, clientId, botId) {
    $scope.clientId = clientId;
    $scope.data = {};
    $scope.data.clientId = clientId;
    $scope.data.clientId = botId;
    
    $scope.close = function() {
      $uibModalInstance.dismiss();
    }

    $scope.login = function(data) {
      buildModel('botlogin', data).post(function(data) {
	//$scope.bodypretty = $scope.pretty(data);
	console.log(data);
      });

      $scope.close();
    }
  }

  app.controller("wechatLoginCtrl", wechatLoginCtrl);
  wechatLoginCtrl.$inject = ["$scope", "$uibModalInstance", "toastr", "buildModel", "buildPromise", "tools", "clientId", "botId"];
  function wechatLoginCtrl($scope, $uibModalInstance, toastr, buildModel, buildPromise, tools, clientId, botId) {
    $scope.clientId = clientId;
    $scope.data = {};
    $scope.data.clientId = clientId;
    $scope.data.botId = botId;
    
    $scope.close = function() {
      $uibModalInstance.dismiss();
    }

    $scope.login = function(data) {
      buildModel('botlogin', data).post(function(data) {
	//$scope.bodypretty = $scope.pretty(data);
	console.log(data);
      });

      $scope.close();
    }
  }

  app.controller("botActionCtrl", botActionCtrl);
  botActionCtrl.$inject = ["$http", "$scope", "$uibModalInstance", "toastr", "buildModel", "buildModelResId", "buildPromise", "tools", "clientId", "login"];
  function botActionCtrl($http, $scope, $uibModalInstance, toastr, buildModel, buildModelResId, buildPromise, tools, clientId, login) {
    $scope.clientId = clientId;
    $scope.data = {};
    $scope.data.clientId = clientId;
    $scope.data.login = login;

    $scope.close = function() {
      $uibModalInstance.dismiss();
    }

    let url = "/botaction/" + $scope.data.login;
    
    $scope.sendAction = function(data) {
      $http({
	method: 'POST',
	url: url,
	data: JSON.stringify(data)
      })
	.then(function (success) {
	  console.log(success);
	}, function(error) {
	  console.log(error);
	});

      $scope.close();
    }    
  }
})();


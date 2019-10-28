(function() {
  "use strict"
  app.controller("clientslistCtrl", clientslistCtrl);
  clientslistCtrl.$inject = ["$scope", "$modal", "toastr", "buildModel",
		             "buildPromise", "tools", "buildModelResId"];
  function clientslistCtrl($scope, $modal, toastr,
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
    
    $scope.initView = (data) => {
      $scope.body = data.body
    }
    
    $scope.refresh = () => {
      buildPromise(buildModel('consts'))
	.then((data) => {
	  $scope.consts = data.body
	  
	  buildPromise(buildModel('clients'))
	    .then((data) => {
	      $scope.initView(data);
	    })
	})
    }

    $scope.refresh()

    $scope.shutdown = (row) => {
      buildPromise(buildModelResId('clients', row.clientId + '/shutdown').update((data) => {
        toastr.success(data, '关闭成功')
      }))
    }
  }
})();

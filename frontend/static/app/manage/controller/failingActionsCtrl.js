(function() {
  "use strict"
  app.controller("failingActionsCtrl", failingActionsCtrl);
  failingActionsCtrl.$inject = ["$scope", "$modal", "toastr", "buildModel",
		                "buildPromise", "tools", "buildModelResId"];
  function failingActionsCtrl($scope, $modal, toastr,
			      buildModel, buildPromise, tools, buildModelResId) {
    $scope.body = {};
    
    $scope.tsToString = function(unix_timestamp) {
      if(unix_timestamp === undefined) { return "" }
      
      var date = new Date(unix_timestamp*1000);

      var year = date.getFullYear();
      var month = "0" + (date.getMonth()+1);
      var datestr = "0" + date.getDate();
      var hours = "0" + date.getHours();
      var minutes = "0" + date.getMinutes();
      var seconds = "0" + date.getSeconds();
      
      return year+'-'+month.substr(-2)+'-'+datestr.substr(-2)+' ' +
	hours.substr(-2)+':'+minutes.substr(-2)+':'+seconds.substr(-2);
    }

    $scope.recoverAction = (row) => {
      buildPromise(buildModel('botaction/recoveraction', row).update((data) => {
        $scope.refresh()
      }))
    }

    $scope.recoverClient = (row) => {
      buildPromise(buildModel('botaction/recoverclient', row).update((data) => {
        $scope.refresh()
      }))
    }
    
    $scope.initView = (data) => {
      $scope.body = data.body
    }
    
    $scope.refresh = () => {
      buildPromise(buildModel('botaction/failing'))
	.then((data) => {
	  $scope.initView(data);
	})
    }

    $scope.refresh()
  }
})();

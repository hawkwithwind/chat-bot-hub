(function() {
  "use strict"
  app.controller("chatusersCtrl", chatusersCtrl)
  chatusersCtrl.$inject = ["$scope", "$modal", "toastr", "buildModel",
		           "buildPromise", "tools", "buildModelResId"]
  function chatusersCtrl($scope, $modal, toastr,
			 buildModel, buildPromise, tools, buildModelResId) {
    $scope.body = {}

    function jump(i) {
      buildPromise(
        buildModel('chatusers'),
        {
          'page': i-1,
          'pagesize': $scope.paging.pagesize ? $scope.paging.pagesize : 100,
        })
        .then((data) => {
          $scope.initView(data)
        })
    }
    
    $scope.initView = (data) => {
      $scope.body = data.body
      $scope.paging = data.paging
      $scope.paging.pagerange = []      
      for (var i=0;i<$scope.paging.pagecount;i++) {
	$scope.paging.pagerange.push(i+1)
      }

      $scope.paging.jump = jump
    }

    $scope.paging.jump = jump
    
    $scope.refresh = () => {
      buildPromise(buildModel('consts'))
	.then((data) => {
	  $scope.consts = data.body
          $scope.paging.jump($scope.paging.page ? $scope.paging.page : 1)
	})
    }

    $scope.refresh()
  }  
})()




	  

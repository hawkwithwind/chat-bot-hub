(function() {
  "use strict"
  app.controller("chatgroupsCtrl", chatgroupsCtrl)
  chatgroupsCtrl.$inject = ["$scope", "$modal", "toastr", "buildModel",
		           "buildPromise", "tools", "buildModelResId"]
  function chatgroupsCtrl($scope, $modal, toastr,
			  buildModel, buildPromise, tools, buildModelResId) {
    $scope.body = {
      criteria: {},
    }
    $scope.paging = {}
    $scope.criteria = {
      groupname: '',
      nickname: '',
      type: '',
    }

    $scope.search = () => {
      $scope.body.criteria = $scope.criteria
      $scope.paging.page = 1
      $scope.initView()
    }
    
    $scope.initView = () => {
      $scope.paging.jump = (i) => {
        buildPromise(
          buildModel('chatgroups'),
          {
            'groupname': $scope.body.criteria.groupname,
            'nickname': $scope.body.criteria.nickname,
            'type': $scope.body.criteria.type,
            'page': i,
            'pagesize': $scope.paging.pagesize ? $scope.paging.pagesize : 100,
          })
          .then((data) => {
            $scope.body = data.body
            
            $scope.paging.page = data.paging.page
            $scope.paging.pagesize = data.paging.pagesize
            $scope.paging.pagecount = data.paging.pagecount
            
            $scope.paging.pagerange = []      
            for (var i=0;i<$scope.paging.pagecount;i++) {
	      $scope.paging.pagerange.push(i+1)
            }
          })
      }

      buildPromise(buildModel('consts'))
	.then((data) => {
	  $scope.consts = data.body
          $scope.paging.jump($scope.paging.page ? $scope.paging.page : 1)
	})
    }

    $scope.initView()
  }
})()




	  

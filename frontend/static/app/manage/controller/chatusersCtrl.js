(function() {
  "use strict"
  app.controller("chatusersCtrl", chatusersCtrl)
  chatusersCtrl.$inject = ["$scope", "$modal", "toastr", "buildModel",
		           "buildPromise", "tools", "buildModelResId"]
  function chatusersCtrl($scope, $modal, toastr,
			 buildModel, buildPromise, tools, buildModelResId) {
    $scope.body = {
      criteria: {},
    }
    $scope.paging = {}
    $scope.criteria = {
      username: '',
      nickname: '',
      type: '',
      botid: '',
    }

    $scope.search = () => {
      $scope.body.criteria = $scope.criteria
      $scope.paging.page = 1
      $scope.initView()
    }
    
    $scope.initView = () => {
      $scope.paging.jump = (i) => {
        buildPromise(
          buildModel('chatusers'),
          {
            'username': $scope.body.criteria.username,
            'nickname': $scope.body.criteria.nickname,
            'type': $scope.body.criteria.type,
            'botid': $scope.body.criteria.botid,
            'page': i,
            'pagesize': $scope.paging.pagesize ? $scope.paging.pagesize : 100,
          })
          .then((data) => {
            $scope.body = data.body
            
            $scope.paging.page = data.paging.page
            $scope.paging.pagesize = data.paging.pagesize
            $scope.paging.pagecount = data.paging.pagecount

             if($scope.paging.pagecount > 20) {
              let pageset = new Set()
              for(var i=0; i<5; i++){
                pageset.add(i+1)
                pageset.add(parseInt($scope.paging.pagecount, 10)-i)
              }

               for(var i=0; i<3; i++) {
                 let before = parseInt($scope.paging.page, 10) + i
                 let after  = parseInt($scope.paging.page, 10) - i
                 for(var j in [before, after]) {
                   if(j >= 0 && j < parseInt($scope.paging.pagecount, 10)) {
                     pageset.add(j+1)
                   }
                 }                
               }

               $scope.paging.pagerange = [...pageset].sort()
             } else {
               $scope.paging.pagerange = []
               for (var i=0;i<$scope.paging.pagecount;i++) {
	         $scope.paging.pagerange.push(i+1)
               }
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




	  

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
              let pagehead = []
              let pagetail = []

              const headpad = 3
              const tailpad = 3
              const middlepad = 3
              
              for(var i=0; i<headpad; i++){
                pagehead.push(i+1)                
              }
              for(var i=0; i<tailpad; i++){
                pagetail.push(parseInt($scope.paging.pagecount, 10)-1)
              }
              
              let p = parseInt($scope.paging.page, 10) + 1
              let before = p - middlepad
              if(before < pagehead[pagehead.length-1]) {
                before = pagehead[pagehead.length-1]
              }
              
              let after  = p + middlepad
              if(after > pagehead[0]) {
                after = pagehead[0]
              }

              let middleset = new Set()
              for(var j = before;j <= after; j++) {
                middleset.add(j)
              }
              console.log(before, after, [...middleset])
              
              $scope.paging.pagerange = pagehead
              console.log($scope.paging.pagerange)
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




	  

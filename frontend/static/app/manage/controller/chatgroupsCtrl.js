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
            if($scope.paging.pagecount > 20) {
              const headpad = 3
              const tailpad = 3
              const middlepad = 3
              
              for(var i=0; i<headpad; i++){
                $scope.paging.pagerange.push(i+1)                
              }
                            
              let p = parseInt($scope.paging.page, 10)
              let before = p - middlepad
              if(before <= headpad) {
                before = headpad + 1
              } else {
                $scope.paging.pagerange.push('D1')
              }
              let afterflag = false
              let after  = p + middlepad
              if(after >= parseInt($scope.paging.pagecount) - tailpad) {
                after = parseInt($scope.paging.pagecount) - tailpad -1
              } else {
                afterflag = true
              }
              for(var j = before;j <= after; j++) {
                $scope.paging.pagerange.push(j)
              }
              if(afterflag) {
                $scope.paging.pagerange.push('D2')
              }
              for(var i=0; i<tailpad; i++){
                $scope.paging.pagerange.push(parseInt($scope.paging.pagecount, 10)-tailpad+i)
              }
              console.log($scope.paging.pagerange)
            } else {
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




	  

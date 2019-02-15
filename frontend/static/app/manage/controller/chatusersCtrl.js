(function() {
  "use strict"
  app.controller("chatusersCtrl", chatusersCtrl)
  chatusersCtrl.$inject = ["$scope", "$modal", "toastr", "buildModel",
		           "buildPromise", "tools", "buildModelResId"]
  function chatusersCtrl($scope, $modal, toastr,
			 buildModel, buildPromise, tools, buildModelResId) {
    $scope.body = {}
    $scope.paging = {
      page: 1,
      pagesize: 100,
    }
    
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
      $scope.paging = data.paging
      $scope.paging.pagerange = []
      for (var i=0;i<$scope.paging.pagecount;i++) {
	$scope.paging.pagerange.push(i+1)
      }
    }

    $scope.paging.jump = (i) => {
      console.log('page jump %d', i)
      buildPromise(
        buildModel('chatusers'),
        {
          'page': i-1,
          'pagesize': $scope.paging.pagesize
        })
        .then((data) => {
          $scope.initView(data)
        })
    }

    $scope.refresh = () => {
      buildPromise(buildModel('consts'))
	.then((data) => {
	  $scope.consts = data.body
          $scope.paging.jump($scope.paging.page)
	})
    }

    $scope.refresh()
  }  
})()




	  

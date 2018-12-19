(function() {
  "use strict"
  app.controller("filterslistCtrl", filterslistCtrl);
  filterslistCtrl.$inject = ["$scope", "$modal", "toastr", "buildModel",
			     "buildPromise", "tools", "buildModelResId"];
  function filterslistCtrl($scope, $modal, toastr,
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
      $scope.body = data.body;
    }

    $scope.refresh = () => {    
      buildPromise(buildModel('filters'))
	.then((data) => {
	  $scope.initView(data)	  
	}).catch((err) => {
	  toastr.error(err, '获取过滤器失败')
	})
    }

    $scope.refresh()

    $scope.createFilter = (row) => {
      $modal.open({
	templateUrl: 'createFilterTemplate',
	controller: createFilterCtrl,
      }).then(() => {
	$scope.refresh()
      })
    }

    $scope.editFilter = (row) => {
      $modal.open({
	templateUrl: 'editFilterTemplate',
	controller: editFilterCtrl,
	resolve: {
	  name: () => row.name,
	  filterId: () => row.filterId,
	  type: () => row.type,
	  body: () => row.body,
	  next: () => row.next,
	}
      }).then(() => {
	$scope.refresh()
      })
    }

  }

  app.controller('createFilterCtrl', createFilterCtrl)
  createFilterCtrl.$inject = ["$http", "$scope", "$uibModalInstance", "toastr", "buildModel", "buildModelResId", "buildPromise", "tools"]
  function createFilterCtrl($http, $scope, $uibModalInstance, toastr, buildModel, buildModelResId, buildPromise, tools) {
    $scope.data = {}

    $scope.close = () => {
      $uibModalInstance.dismiss()
    }

    let url = "/filters"
    $scope.createFilter = (data) => {
      $http({
	method: 'POST',
	url: url,
	data: data
      }).then((success) => {
	toastr.success($scope.data.name, '创建成功')
      }, (error) => {
	toastr.error(error, '创建失败')
      })

      $scope.close()
    }
  }

  app.controller('editFilterCtrl', editFilterCtrl)
  editFilterCtrl.$inject = ["$http", "$scope", "$uibModalInstance", "toastr", "buildModel", "buildModelResId", "buildPromise", "tools", "filterName", "filterType", "body", "next", "filterId"]
  function editFilterCtrl($http, $scope, $uibModalInstance, toastr, buildModel, buildModelResId, buildPromise, tools, name, type, body, next, filterId) {
    $scope.data = {
      filterId: filterId,
      name: name,
      type: type,
      body: body,
      next: next,      
    }

    $scope.close = () => {
      $uibModalInstance.dismiss()
    }

    let url = "/filters/" + $scope.data.filterId
    $scope.saveFilter = (data) => {
      $http({
	method: 'PUT',
	url: url,
	data: data,
      }).then((success)=> {
	toastr.success(success, '编辑成功')
      }, (error)=>{
	toastr.error(error, '编辑失败')
      })

      $scope.close()
    }    
  }
  
  
})();

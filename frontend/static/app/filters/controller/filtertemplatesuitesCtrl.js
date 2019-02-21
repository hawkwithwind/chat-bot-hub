(function() {
  "use strict"
  app.controller("filtertemplatesuitesCtrl", filtertemplatesuitesCtrl);
  filtertemplatesuitesCtrl.$inject = ["$scope", "$modal", "toastr", "buildModel",
			              "buildPromise", "tools", "buildModelResId"];
  function filtertemplatesuitesCtrl($scope, $modal, toastr,
			            buildModel, buildPromise, tools, buildModelResId) {
    $scope.body = {};

    $scope.initView = (data) => {
      $scope.body = data.body;
    }

    $scope.refresh = () => {    
      buildPromise(buildModel('filtertemplatesuites'))
	.then((data) => {
	  $scope.initView(data)
	}).catch((err) => {
	  toastr.error(err, '获取过滤器套件模板失败')
	})
    }

    $scope.refresh()

    $scope.createFTSuite = (row) => {
      $modal.open({
	templateUrl: 'createFTSuiteTemplate',
	controller: createFTSuiteCtrl,
      }).then(() => {
	$scope.refresh()
      })
    }

    $scope.editFTSuite = (row) => {
      $modal.open({
	templateUrl: 'editFTSuiteTemplate',
	controller: editFTSuiteCtrl,
	resolve: {
	  name: () => row.name,
	  id: () => row.id
	}
      }).then(() => {
	$scope.refresh()
      })
    }

  }

  app.controller('createFilterTSCtrl', createFilterTSCtrl)
  createFilterTSCtrl.$inject = ["$http", "$scope", "$uibModalInstance", "toastr", "buildModel", "buildModelResId", "buildPromise", "tools"]
  function createFilterTSCtrl($http, $scope, $uibModalInstance, toastr, buildModel, buildModelResId, buildPromise, tools) {
    $scope.data = {}

    $scope.close = () => {
      $uibModalInstance.dismiss()
    }

    let url = "/filtertemplatesuites"
    $scope.newFTSuite = (data) => {
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

  app.controller('editFTSuiteCtrl', editFTSuiteCtrl)
  editFTSuiteCtrl.$inject = ["$http", "$scope", "$uibModalInstance", "toastr", "buildModel", "buildModelResId", "buildPromise", "tools", "name", "id"]
  function editFTSuiteCtrl($http, $scope, $uibModalInstance, toastr, buildModel, buildModelResId, buildPromise, tools, name, id) {
    $scope.data = {
      id: id,
      name: name,
    }

    $scope.close = () => {
      $uibModalInstance.dismiss()
    }

    let url = "/filtertemplatesuites/" + $scope.data.id
    $scope.saveFTSuite = (data) => {
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

    let addfturl = "/filtertemplates"
    $scope.addFilterTemplate = (data) => {
      $http({
        method: 'POST',
        url: addfturl,
        data: {
          index: data.tempIndex,
          type: data.tempType,
          name: data.tempName,
        },
      }).then((success) => {
        toastr.success(success, '插入成功')
      }, (error) => {
        toastr.error(error, '插入失败')
      })
    }
  }
  
  
})();

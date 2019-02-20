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

    $scope.createFilter = (row) => {
      $modal.open({
	templateUrl: 'createFilterTSTemplate',
	controller: createFilterTSCtrl,
      }).then(() => {
	$scope.refresh()
      })
    }

    $scope.editFilter = (row) => {
      $modal.open({
	templateUrl: 'editFilterTSTemplate',
	controller: editFilterTSCtrl,
	resolve: {
	  name: () => row.name,
	  filterTemplateSuiteId: () => row.filterTemplateSuiteId
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
    $scope.createFilterTS = (data) => {
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

  app.controller('editFilterTSCtrl', editFilterTSCtrl)
  editFilterTSCtrl.$inject = ["$http", "$scope", "$uibModalInstance", "toastr", "buildModel", "buildModelResId", "buildPromise", "tools", "name", "filterTemplateSuiteId"]
  function editFilterTSCtrl($http, $scope, $uibModalInstance, toastr, buildModel, buildModelResId, buildPromise, tools, name, filterTemplateSuiteId) {
    $scope.data = {
      filterId: filterId,
      name: name,
    }

    $scope.close = () => {
      $uibModalInstance.dismiss()
    }

    let url = "/filtertemplatesuites/" + $scope.data.filterTemplateSuiteId
    $scope.saveFilterTS = (data) => {
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

(function (argument) {
	app
		.directive('selectSku', selectSku);

		function selectSku() {
			var directive = {
				restrice: 'AE',
				replace: true,
				template: '<div class="input-group">'+
		                '<input type="text" ng-disabled="data.requestLoading" class="form-control" ng-model="data.code" ng-enter="event.queryForCode()" placeholder="请输入商品编码,回车添加" />'+
		                '<span class="input-group-addon cursor-pointer" ng-click="event.selectSku()">...</span>' +
	                '</div>',
	            controller: selectSkuCtrl,
	            scope: {
	            	onSelect: '&',
	            	options: '='
	            },
				link: link

			};
			selectSkuCtrl.$inject = ['$scope', 'productModel', 'tools'];

			return directive;

			function selectSkuCtrl($scope, productModel, tools) {
				$scope.data = {
					code: '',
					requestLoading: false
				};
				$scope.event = {
					selectSku: selectSku,
					queryForCode: queryForCode
				};
				$scope.options = angular.extend({inputConfig: {}, modalConfig: {}}, $scope.options);

				function selectSku() {
					if (angular.isFunction($scope.options.modalConfig.onBeforeModalOpen)) {
						var result = $scope.options.modalConfig.onBeforeModalOpen($scope.data.code);
						if (result === false) {
							return false;
						}
					}
					$scope.options.queryCode = $scope.data.code;
					var config = {
						resolve: {
							options: function() {
								return $scope.options;
							}
						}
					};
					if ($scope.options.modalConfig && $scope.options.modalConfig.size) {
						config.size = $scope.options.modalConfig.size;
					}
					var modalInstance = tools.selectProduct(config);

		            modalInstance.result.then(function(data) {
		                $scope.onSelect({data: data});
		            });
				}
				function queryForCode() {
					if (!valid()) {
						return false;
					}

					var modelKey = ($scope.options.inputConfig.modelType) || 'productList';

					$scope.data.requestLoading = true;

		            // 开始请求
		            var result =  getParams();

		            productModel[modelKey].query(result, function(d) {
		                $scope.data.requestLoading = false;

		                if (d.data == null || d.data.length === 0) {
		                    tools.toastr.error("没找到指定的商品,请检查商品编号是否正确。");
		                    return false;
		                }
		                $scope.data.code = '';

		                if (angular.isFunction($scope.onSelect)) {
		                	var resultData = null;
		                	if (!angular.isArray(d.data)) {
		                		resultData = [d.data];
		                	} else {
								resultData = d.data;
		                	}
		                	$scope.onSelect({data: resultData});
		                }
		            }, function() {
		                $scope.data.requestLoading = false;
		            });
				}
				function valid() {
					if (angular.isFunction($scope.options.inputConfig.onBefore)) {
						var result = $scope.options.inputConfig.onBefore($scope.data.code);
						if (!result) {
							return false;
						}
					}
					return true;
				}
				function getParams() {
					var queryData = {};
					for (var key in $scope.options.inputConfig.queryFieldName) {
						queryData[$scope.options.inputConfig.queryFieldName[key]] = $scope.data[key];
					}
					queryData.code = $scope.data.code;
					return angular.extend({}, queryData, $scope.options.params);
				}
			}
			function link(scope, elem, attr) {

			}
		}
})();
(function() {
	"use strict";
	//同意表格
	app.directive('gridTable', ['tools', '$compile',
		function(tools, $compile) {
		return {
			restrict: 'E',
			replace: true,
			transclude: true,
			templateUrl:  'static/app/common/views/tableTemplate.html',
			scope: {
				grid_options: '=gridOptions',
				grid_data: '=gridData',
				global_data: '=globalData',
				total: '=totalPage',
				page_change: '&pageChange',
				cur_page: '=page',
				order_by: '&orderBy',
				page_offset: '=pageOffset',
				repeat_class: '@repeatClass',
				checkbox: '=',
				checkbox_show: '@checkboxShow',
				repeat_init: '&repeatInit',
				hide_col: '@hideCol',
				scroll: '@',
				set_iden:'@setIden',
				sort_name:'&sortName'
			},
			compile: function(elem, attrs) {
				var dataListStr = attrs['gridData'],
					animate = 'slide-down',
					templateHtml = $('#' + attrs['templateId']).html(),
					$table = elem.find('table'),
					repeatClass = attrs['repeatClass'] ? ('ng-class="' + attrs['repeatClass'] + '"') : '',
					hasCheckbox = attrs['checkbox'] || attrs['checkboxId'],
					setIden =  attrs['setIden']?('dataIden="'+ attrs['setIden'] +'"'):'',
					checkboxShow = '';
				//calcGridHeight();
				bulidHtmlTemplate();

				slimscroll();
				function slimscroll() {
					if (attrs['scroll']) {
						elem.find('.table-responsive').attr('slimscroll', attrs['scroll']);
					}
				}
				function bulidHtmlTemplate() {
					if (!attrs['totalPage']) {
						elem.find('.grid-pager').remove();
					}
					checkboxShow = hasCheckbox ? bulidCheckboxHtml() : '';

					$table.html('<tbody><tr class="' + animate + ' bind-template" ' + setIden + buildRepeat() + '>'+
					templateHtml + '</tr></tbody>');
					elem.find('.alert').attr('ng-if', 'grid_data.length == 0');
				}
				function buildRepeat() {
					var repeatStr = '',
						initFunc = attrs['repeatInit'],
						filterFunc = attrs['filter'];
					if (initFunc) {
						repeatStr = 'ng-init="repeatInit(item, $index)"';
					}
					if (filterFunc) {
						repeatStr += ' ng-repeat="item in filter_grid_data = (grid_data | filter:' + filterFunc + ')"';
					} else {
						repeatStr += ' ng-repeat="item in (filter_grid_data = grid_data)"';
					}
					return repeatStr;
				}
				function bulidCheckboxHtml() {
					var strId = attrs['checkboxId'] ? ' data-id="[[item.id]]" ' : '',
						strIndex = attrs['checkbox'] ? ' data-index="[[$index]]" ' : '',
						show = attrs['checkboxShow'] ? ' ng-show="' + attrs['checkboxShow'] +  '"' : '',
						rowShow = attrs['rowCheckboxShow'] ? ' ng-show="' + attrs['rowCheckboxShow'] +  '"' : '',
						init = attrs['rowCheckboxShow'] ? ' ng-init="filterCheckbox[item.id] = (' + attrs['rowCheckboxShow'] + ')"' : '';
						templateHtml = "<td" + show + "><div " + rowShow + "><input " + init + " ng-change='checkboxChange()' ng-model='checkedbox[item.id]' type='checkbox' " + strId + strIndex + "'></div></td>" + templateHtml;
					return show;
				}
				return function(scope, elem, attrs) {
					var isFilterRowCheckbox = !!attrs['rowCheckboxShow'];
					init(scope);
					bulidThead(elem, scope);
					scope.global_data = scope.global_data;
					if (scope.grid_options && angular.isObject(scope.grid_options.broadcastData)) {
						for (var name in scope.grid_options.broadcastData) {
							scope[name] = scope.grid_options.broadcastData[name];
							/*if (!angular.isString(scope.grid_options.broadcastData[name])) {
								continue;
							}*/
							(function(name) {
								if (angular.isString(scope.$parent[name]) || angular.isString(scope.$parent[name]) || angular.isNumber(scope.$parent[name])) {
									scope.$watch('$parent.' + name, function(v) {
										scope[name] = v;
									})
								}
							})(name)
						}
					}
					if (hasCheckbox) {
						setTimeout(function() {
							scope.$apply(function() {
								checkboxInit(scope);
							});
						}, 200);
					}

					function checkboxInit(scope) {
						var filter = attrs['filter'];
						scope.filterCheckbox = {};
						if (scope.filter_grid_data) {
							if(filter){
								scope.$watch('filter_grid_data.length', function() {
									scope.checkedbox = {};
									scope.filter_grid_data.forEach(function(v) {
										var id = v.id;
										if (scope.hasCheckedArray) {
											if (scope.filterCheckboxResult.indexOf(id) == -1) {
												scope.checkedbox[id] = false;
											} else {
												scope.checkedbox[id] = true;
											}
										} else {
											scope.checkedbox[id] = false;
										}
									})
									scope.checkboxChange();
								})
							}else{
								scope.$watch('filter_grid_data', function() {
									scope.checkedbox = {};
									scope.filter_grid_data.forEach(function(v) {
										var id = v.id;
										if (scope.hasCheckedArray) {
											if (scope.tmpCheckbox.indexOf(id) == -1) {
												scope.checkedbox[id] = false;
											} else {
												scope.checkedbox[id] = true;
											}
										} else {
											scope.checkedbox[id] = false;
										}
									})
									scope.checkboxChange();
								})
							}
						}
						//记录每次筛选的checkbox状态
						scope.checkboxChange = function() {
							var item, checkedId = [], isAllChecked = true, index = 0;
							for (item in scope.checkedbox) {
								index += 1;

								var itemIndex = scope.filterCheckboxResult.indexOf(parseInt(item));
								if(scope.filterCheckboxResult.indexOf(parseInt(item)) >= 0 && !scope.checkedbox[item]){
									scope.filterCheckboxResult.splice(itemIndex,1);
								}else if(scope.checkedbox[item] && scope.filterCheckboxResult.indexOf(parseInt(item)) == -1){
									scope.filterCheckboxResult.push(parseInt(item));
								}
								if(!scope.checkedbox[item]){
									isAllChecked = false;
								}
							}
							scope.checkbox = scope.filterCheckboxResult;

							if (index == 0) { scope.checkedboxAll = false; return false; }
							scope.checkedboxAll = isAllChecked;
						}

						scope.checkboxAllChange = function() {
							var item, isChecked = scope.checkedboxAll, index = 0;
							for (item in scope.checkedbox) {
								scope.checkedbox[item] = isChecked;
								index += 1;
							}
							if (index == 0) { return false; }
							scope.checkboxChange();
						}

					}
					function init(scope) {
						//所有checkbox对象
						var $tableBox = elem.find('.table-box');
						calcGridHeight(scope);
						scope.checkedbox = {};
						scope.hasCheckedArray = angular.isArray(scope.checkbox);
						scope.tmpCheckbox = scope.filterCheckboxResult = angular.copy(scope.checkbox);
						if (angular.isFunction(scope.order_by)) {
							scope.orderBy = function(name) {
								scope.order_by({name: name});
							}
						}
						if (angular.isFunction(scope.repeat_init)) {
							scope.repeatInit = function(item, index) {
								scope.repeat_init({'item': item, 'index': index});
							}
						}
						if (angular.isFunction(scope.page_change)) {
							scope.pageChange = function(page) {
								$tableBox.scrollTop(0);
								scope.page_change({page: page});
							}
						}
						if(angular.isFunction(scope.sort_name)){
							scope.sortName = function(index){
								scope.sort_name({'index':index});
							}
						}
						scope.toggle = function(status) {
							//if (commonApi.IS_SMALL_SCREEN) {
							//	scope.tableHeight = {'overflow': 'visible', 'height': 'initial'}
							//} else {
							//	//tools.pagerToggle(scope, status, 55);
							//}
						}
						if (scope.grid_options && scope.grid_options.event) {
							scope.event = scope.grid_options.event;
						}
					}
					function calcGridHeight(scope) {
						if (true) {
							var height = attrs['pageOffset'];
							if (isNaN(height)) {
								height = scope.page_offset;
							}
							height = (parseInt(height) ? parseInt(height) : 0);
							elem.css({'height': 'calc(100% - ' + (height) + 'px)'});
						}
					}
					function bulidThead(elem, scope) {
						var thead = '<thead{0}><tr>{1}</tr></thead>', 
							colgroup = "",
							colHtml = "",
							html = "", 
							className = "",
							isSort = false;
						if (!angular.isArray(scope.grid_options.fields)) {
							return false;
						}
						scope.grid_options.fields.forEach(function(v,j) {
							if (scope.hide_col == "1" && v.hideField) {
								return;
							}
							var sortStr = '';
							className = '';
							if (v.sort) {
								sortStr = ' field-name="' + v.sort + '"';
								isSort = true;
							}
							if (v.width) {
								colHtml += '<col width="' + v.width + '" />';
							}
							if (v.className) {
								className = ' class="' + v.className + '" ';
							}
							html += '<th' + sortStr + className + '>' + '<span ng-click=sortName('+j+')>'+ v.field + '</span>'+ '</th>';
						})
						if (hasCheckbox) {
							html = '<th ' + checkboxShow + '><input ng-model="checkedboxAll" ng-change="checkboxAllChange()" type="checkbox"></th>' + html;
							colHtml =  '<col width="2%" ' + checkboxShow + ' />' + colHtml;
						}
						thead = thead.replace('{0}', isSort ? ' sort-head=orderBy(name)' : '');
						thead = thead.replace('{1}', html);
						if (colHtml) {
							colgroup = '<colgroup>' + colHtml + '</colgroup>';
							thead = colgroup + thead;
						}
						elem.find('.custom-table').prepend($compile(thead)(scope));
					}
				}
			}
		}
	}])
	/*
	$scope.gridOption = {
		data: 
		fields:	[
			{field: "账号名称"},
			{field: "累计积分", sort: 'total_point'},
			{field: "当前总积分", sort: 'current_point'},
			{field: "已兑换积分", sort: 'exchange_point'},
			{field: "已冻结积分"}
		],
		event: {
			aa : function(item, index) {
				console.log(item, index)
			}
		},
		broadcastData: {
			searchObj: $scope.searchObj
		}
	}

	
	<grid-table 
		grid-options="gridOption"
		grid-data="userPointData.userpoint_list" 
		global-data="globalData"
		template-id="userPointTemplate" 
		total-page='userPointData.total_page' 
		page-change="pageChange(page)" 
		page="searchObj.page"
		order-by="orderBy(name)"
		page-offset="52"
		repeat-class="{'restaurant-processing': item.audit_status == 30}"
		checkbox="checkIndex"
		checkboxShow="process==0"
		scroll="300px"
		filter=""
		>
	</grid-table>
						
	<template id="userPointTemplate">
		<td><a href="javascript:;" ng-click="event.aa(item, $index)">[[:: item.username]]</a></td>
		<td>[[:: item.total_point]]</td>
		<td>[[:: item.current_point]]</td>
		<td>[[:: item.exchange_point]]</td>
		<td>[[:: item.current_point]]</td>
	</template>
	*/

})();
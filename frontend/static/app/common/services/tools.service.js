(function () {
  app.factory('tools', tools);
  // 集合基础服务
  tools.$inject = ['toastr', 'block', 'notification', '$window', '$uibModal', 'uploadRequest', '$timeout'];

  function tools(toastr, block, notification, $window, $uibModal, uploadRequest, $timeout) {
    var moment = $window.moment;
    return {
      $timeout: $timeout,
      toastr: toastr,
      block: block,
      notification: notification,
      modal: $uibModal,
      uploadRequest: uploadRequest,
      filterEmptyProp: filterEmptyProp,
      setSearchPage: setSearchPage,
      clearFilter: clearFilter,
      clearExportUrl: clearExportUrl,
      moment: moment,
      getDiffDate: getDiffDate,
      getDiffMonth: getDiffMonth,

      getItemInArray: getItemInArray,
      addPlaceholder: addPlaceholder,

      selectProduct: selectProduct,
      getItemById: getItemById,
      getStatusList: getStatusList
    };
    function clearExportUrl(data, key) {
      $timeout(function() {
        data[key || 'exportExcelUrl'] = "";
      });
    }
    function getStatusList(statuses) {
      var arr = [];
      angular.forEach(statuses, function(val, index) {
        if (val) {
          arr.push(index);
        }
      });
      return arr.join(',');
    }
    function clearFilter(sourceObj, extendObj) {
      var newData = {};
      if (sourceObj) {
	if (sourceObj.newPage) {
	  newData.curPage = sourceObj.newPage;
	} 
	if (sourceObj.pageSize) {
	  newData.pageSize = sourceObj.pageSize;
	}
      }
      return angular.extend(newData, extendObj);
    }
    function selectProduct(_resolve) {
      var config = {
	templateUrl: 'catalog/product-list.html',
	controller: 'ProductModalCtrl',
	size: 'lg',
	resolve : {
	  params: function() {
	    return {};
	  }
	}
      };
      var _config = angular.extend(config, _resolve);

      return $uibModal.open(_config);
    }

    function addPlaceholder(list, item) {
      var defaultPlaceholder = {id: '', name: '--请选择--'};
      list = angular.isArray(list) ? angular.copy(list) : [];
      if (angular.isString(item)) {
	defaultPlaceholder[item] = defaultPlaceholder.name;
	item = defaultPlaceholder;
      } else if (angular.isObject(item)) {
	item = item;
      } else {
	item = defaultPlaceholder;
      }

      list.unshift(item);
      return list;
    }
    function getItemInArray (arr, val, prop) {
      var prop = prop || 'id';

      for (var i = 0, len = arr.length; i < len; i++) {
	if (arr[i][prop] == val) {
	  return arr[i];
	}
      }
      return arr[0];
    }

    function getDiffDate(diff, isEnd, format) {
      diff = diff || 0;
      var now = getDate(isEnd);
      now.date(now.date() + diff);
      return formatMoment(now, format);
    }
    function getDiffMonth(diff, isEnd, format) {
      diff = diff || 0;
      var now = getDate(isEnd);
      now.month(now.month() + diff);
      return formatMoment(now, format);
    }

    function formatMoment(instance, format) {
      return instance.format( format || "YYYY-MM-DD HH:mm:ss");
    }

    function getDate(isEnd) {
      var date = isEnd ? getDayEnd() : getDayStart();
      return date;
    }
    //获取当天开始时间 00:00:00
    function getDayStart() {
      var start = moment( moment().format('YYYY-MM-DD'));
      return start;
    }
    function getDayEnd() {
      var end = getDayStart();
      // 加一天
      end.date(end.date() +1);
      // 减一秒
      end.second(end.second() - 1);
      return end;
    }
    function filterEmptyProp(obj, prop) {
      for (var key in obj) {
	if (obj.hasOwnProperty(key) && (obj[key] === '' || obj[key] === null || obj[key] === undefined)) {
	  delete obj[key];
	}
      }
      if (angular.isArray(prop)) {
	prop.forEach(function (v) {
	  removeProp(obj, v);
	});
      } else if (angular.isString(prop)) {
	removeProp(obj, prop);
      }
      return obj;
    }
    function setSearchPage(obj, isSearch) {
      if (isSearch) {
	obj.curPage = 1;
      }
      return obj;
    }
    function removeProp (obj, key) {
      delete obj[key];
    }

    // 根据Id取得item
    function getItemById(list, id, key) {
      key = key || "id";
      for(var i = 0; i < list.length; i++) {
	if (list[i][key] == id) {
	  return list[i];
	}
      }
    }
  }
})();

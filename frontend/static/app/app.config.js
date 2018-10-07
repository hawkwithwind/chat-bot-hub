(function() {
  "use strict"
  app.config(config);
  
  config.$inject = ['cfpLoadingBarProvider', '$ocLazyLoadProvider', 'JS_REQUIRES', '$httpProvider', '$uibTooltipProvider', '$uibModalProvider'];
  function config(cfpLoadingBarProvider, $ocLazyLoadProvider, jsRequires, $httpProvider, $uibTooltipProvider, $uibModalProvider) {
    // loadingbar
    cfpLoadingBarProvider.includeBar = true;
    cfpLoadingBarProvider.includeSpinner = false;

    // ocLazyLoad
    $ocLazyLoadProvider.config({
      debug: false,
      events: true,
      cache: false,
      modules: jsRequires.modules
    });
    //格式化成 默认的 formdata类型的数据
    $httpProvider.defaults.headers.post['Content-Type'] = 'application/x-www-form-urlencoded;charset=utf-8';
    $httpProvider.defaults.headers.put['Content-Type'] = 'application/x-www-form-urlencoded;charset=utf-8';

    $httpProvider.defaults.withCredentials = true;
    $httpProvider.defaults.transformRequest = [function(data) {
      return angular.isObject(data) && String(data) !== '[object File]' ? param(data) : data;
    }];
    // 添加拦截器
    $httpProvider.interceptors.push(interceptors);

    // tooltip
    $uibTooltipProvider.options({appendToBody: true});
    $uibTooltipProvider.setTriggers({'myMouseenter': 'windowClick'}); 
    $uibTooltipProvider.options({ popoverMode: 'single'});
    // tooltip
    //$uibPopoverProvider.options({appendToBody: true});
    $uibModalProvider.options = {
      animation: true,
      backdrop: false, //can also be false or 'static'
      keyboard: true
    };

  }
  function param(obj) {
    var query = '', name, value, fullSubName, subName, subValue, innerObj, i;
    for(name in obj) {
      value = obj[name];
      if(value instanceof Array) {
	for(i=0; i < value.length; ++i) {
	  subValue = value[i];
	  fullSubName = name + '[' + i + ']';
	  innerObj = {};
	  innerObj[fullSubName] = subValue;
	  query += param(innerObj) + '&';
	}
      } else if(value instanceof Object) {
	for(subName in value) {
	  subValue = value[subName];
	  fullSubName = name + '.' + subName; //+ '[' + subName + ']';
	  innerObj = {};
	  innerObj[fullSubName] = subValue == null ? '' : subValue;

	  query += param(innerObj) + '&';
	}
      } else if(value !== undefined && value !== null)
	query += encodeURIComponent(name) + '=' + encodeURIComponent(value) + '&';
    }
    return query.length ? query.substr(0, query.length - 1) : query;
  }
  interceptors.$inject = ['$window', '$q', 'toastr', 'block', '$location', '$timeout' , 'cfpLoadingBar'];
  function interceptors($window, $q, toastr, block, $location, $timeout, cfpLoadingBar) {
    var isShowError = true,
	$body = $('body'),
	ajaxQueue = [],
	blockIdQueue = {};
    function showError(msg) {
      if (isShowError) {
	toastr.error(msg);
	isShowError = false;
	timeoutShowError();
	cfpLoadingBar.complete();
      }
    }
    function timeoutShowError() {
      $timeout(function() {
	isShowError = true;
      }, 1000);
    }
    function responseErrorLocation(url) {
      $timeout(function() {
	$window.location.href = url;
      }, 100);
    }
    function removeBlock(config) {
      if (ajaxQueue.length === 0) {
	for (var key in blockIdQueue) {
	  if (blockIdQueue.hasOwnProperty(key)){
	    block.unblockUI(key);
	  }
	}
	blockIdQueue = {};
      }
    }
    return {
      request: function (config) {
	// 全局ajax 超时时间
	//config.timeout = 10000;
	if (config.url.indexOf('http') > -1) {
	  if (!(config.params && config.params.noBlock)) {
	    if (config.params && config.params.blockId) {
	      config.blockId = config.params.blockId;
	    } else {
	      if ($body.children('.modal').length === 0) {
		config.blockId = '#container';
	      } else {
		config.blockId = 'body';
	      }
	    }
	    blockIdQueue[config.blockId] = 1;
	    block.blockUI({target: config.blockId});
	    ajaxQueue.push(config.url);
	  }
	}

	// set jwt token bearer
        var token=$window.sessionStorage.getItem('user_token');

	if (token) {
          //set authorization header
          config.headers['Authorization'] = 'Bearer '+token;
	}
	return config;
      },
      response: function (response) {
	
	ajaxQueue.splice(ajaxQueue.indexOf(response.config.url), 1);
	var data = response.data;
	removeBlock();

	if (response.config.url === 'login' && response.data.body.token) {
          //fetch token
          var token=response.data.body.token;
	  
          //set token
          $window.sessionStorage.setItem('user_token', token);
        }
	return response;
      },
      responseError: function (rejection) {
	ajaxQueue = [];
	removeBlock();
	if (!rejection || !rejection.status) {
	  showError('未知错误');
	  return $q.reject('')
	} else {
	  console.log(rejection.status)
	  if (rejection.status === 404) {
	    showError('找不到指定的url');
	  } else if (rejection.status >= 500) {
	    showError('服务器内部错误');
	  } else if(rejection.status === 403){
	    showError('未登录');
	    $window.sessionStorage.removeItem('user_token');
	    responseErrorLocation("#/app/login/login");
	  } else if(rejection.status === -1){
	    showError('请联系管理员,或没有网络')
	  }
	  return $q.reject(rejection);
	}	
      }
    };
  }
})();

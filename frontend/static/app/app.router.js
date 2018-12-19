(function() {
  'use strict';

  app.config([
    '$stateProvider', '$urlRouterProvider', '$controllerProvider', '$compileProvider', '$filterProvider', '$provide', '$ocLazyLoadProvider', 'JS_REQUIRES', 'VIEW_PATH',
    function ($stateProvider, $urlRouterProvider, $controllerProvider, $compileProvider, $filterProvider, $provide, $ocLazyLoadProvider, jsRequires, VIEW_PATH) {
      
      app.controller = $controllerProvider.register;
      app.directive  = $compileProvider.directive;
      app.filter     = $filterProvider.register;
      app.factory    = $provide.factory;
      app.service    = $provide.service;
      app.constant   = $provide.constant;
      app.value      = $provide.value;


      // 默认路由
      $urlRouterProvider.otherwise("/app/user/login");


      // 路由
      $stateProvider.state('app', {
        url: "/app",
        templateUrl: VIEW_PATH + "layout/app.html",
        resolve:{
          ui: loadSequence('angularMoment', 'ui.ripple', 'ui.datepicker','viewLayout',
			   'ui.maxlength', 'perfect-scrollbar-plugin', 'perfect_scrollbar','ui.tree'
			  ).deps
        },
        ncyBreadcrumb: {
          label: '首页'
        }
      });

      $stateProvider.state('app.user', {
        url: '/user',
        template: '<div ui-view class="fade-in-up" style="height: 100%;overflow-x: hidden;overflow-y: auto;"></div>',
        title: '用户',
        ncyBreadcrumb: {
          label: '用户',
          parent:'app'
        }
      });

      $stateProvider.state('app.user.login', {
        url: "/login",
        templateUrl: VIEW_PATH + "login/views/login.html",
        controller:"loginCtrl",
        ncyBreadcrumb: {
	  label: '登录',
          parent:'app.user'
        }
      });

      $stateProvider.state('app.manage', {
        url: '/manage',
        template: '<div ui-view class="fade-in-up" style="height: 100%;overflow-x: hidden;overflow-y: auto;"></div>',
        title: '管理',
        ncyBreadcrumb: {
          label: '管理',
          parent:'app'
        }
      });

      $stateProvider.state('app.manage.botslist', {
        url: "/botslist",
        templateUrl: VIEW_PATH + "manage/views/botslist.html",
        controller:"botslistCtrl",
        ncyBreadcrumb: {
	  label: '机器人列表',
          parent:'app.manage'
        }
      });

      $stateProvider.state('app.manage.filterslist', {
	url: '/filterslist',
	templateUrl: VIEW_PATH + 'manage/views/filterslist.html',
	controller: 'filtersCtrl',
	ncyBreadcrumb: {
	  label: '过滤器列表',
	  parent: 'app.manage'
	}
      });
      

      // 异步加载需要的ctrl和UI组件
      function loadSequence() {
        var _args = arguments;
        return {
          deps: ['$ocLazyLoad', '$q',
                 function ($ocLL, $q) {
                   var promise = $q.when(1);
                   for (var i = 0, len = _args.length; i < len; i++) {
                     promise = promiseThen(_args[i]);
                   }
                   return promise;

                   function promiseThen(_arg) {
                     if (typeof _arg == 'function') {
                       return promise.then(_arg);
                     } else {
                       return promise.then(function () {
                         var nowLoad = requiredData(_arg);
                         if (!nowLoad)
                           return $.error('没有找到指定的模块 [' + _arg + ']');
                         return $ocLL.load(nowLoad);
                       });
                     }
                   }
                   function requiredData(name) {
                     if (jsRequires.modules) {
                       for (var m in jsRequires.modules) {
                         if (jsRequires.modules[m].name && jsRequires.modules[m].name === name) {
                           return jsRequires.modules[m];
                         }
                       }
                     }
                     return jsRequires.scripts && jsRequires.scripts[name];
                   }
                 }]
        };
      }
    }]);
})();

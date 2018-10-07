'use strict';

angular.module("moduleList", [
  'ngAnimate',
  'ngMessages',
  'ngCookies',
  'ngStorage',
  'ngResource',
  'ui.router',
  'ncy-angular-breadcrumb',
  'oc.lazyLoad',
  'cfp.loadingBar',
  'ui.bootstrap',
  'duScroll',
  'agGrid',
  'ui.toastr',
  'ui.block',
  'ui.notification',
  'templateModal',
  'highcharts-ng'
]);

var app = angular.module('app', ["moduleList"]);
app.run(bootstrap);

bootstrap.$inject = ['$rootScope', '$state', '$stateParams'];

function bootstrap($rootScope, $state, $stateParams) {

    // 添加到全局
    $rootScope.$state = $state;

    $rootScope.$stateParams = $stateParams;

    // 设置全局信息
    $rootScope.app = {
        layout: {
            theme: 'darkblue.min'
        }
    };
    //全局用户信息
    $rootScope.user = {
        name: 'Peter',
        job: 'ng-Dev',
        picture: 'app/img/user/02.jpg'
    };
}

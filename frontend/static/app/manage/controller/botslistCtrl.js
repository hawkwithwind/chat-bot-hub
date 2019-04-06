(function() {
  "use strict"
  app.controller("botslistCtrl", botslistCtrl);
  botslistCtrl.$inject = ["$scope", "$modal", "toastr", "buildModel",
		       "buildPromise", "tools", "buildModelResId"];
  function botslistCtrl($scope, $modal, toastr,
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
      $scope.body = data.body
    }

    $scope.refresh = () => {
      buildPromise(buildModel('consts'))
	.then((data) => {
	  $scope.consts = data.body
	
	  buildPromise(buildModel('bots'))
	    .then((data) => {
	      $scope.initView(data);
	    })
	})
    }

    $scope.refresh()

    $scope.createBot = (row) => {
      $modal.open({
	templateUrl: 'createBotTemplate',
	controller: createBotCtrl,	
      }).then(() => {
	$scope.refresh()
      })
    }

    $scope.scanWechatBot = () => {
      $modal.open({
	templateUrl: 'scanWechatBotTemplate',
	controller: scanWechatBotCtrl,
	resolve: {
	  clientType: () => 'WECHATBOT'
	}
      }).then(() => {
	$scope.refresh()
      })
    }

    $scope.editBot = (row) => {
      $modal.open({
	templateUrl: 'editBotTemplate',
	controller: editBotCtrl,
	resolve: {
	  clientId: () => row.clientId,
	  clientType: () => row.clientType,
	  botName: () => row.botName,
	  login: () => row.login,
	  filterId: () => row.filterId,
	  callback: () => row.callback,
          wxaappId: () => row.wxaappId,
	}
      }).then(() => {
	$scope.refresh()
      })
    }
    
    $scope.showQQLogin = (row) => {
      $modal.open({
	templateUrl: 'loginQQTemplate',
	controller: qqLoginCtrl,
	resolve: {
	  clientId: () => row.clientId
	}
      }).then(() => {
	$scope.refresh()
      })
    }

    $scope.showWechatLogin = (row) => {
      $modal.open({
	templateUrl: 'loginWechatTemplate',
	controller: wechatLoginCtrl,
	resolve: {
	  clientId: () => row.clientId,
	  botId: () => row.botId,
	}
      }).then(() => {
	$scope.refresh()
      })
    }

    $scope.wechatLogin = (row) => {
      buildModel('botlogin', {
	clientId: row.clientId,
	clientType: row.clientType,
	botId: row.botId
      }).post((data) => {
	toastr.success(data, '登录成功')
      })
    }

    $scope.botAction = (row) => {
      $modal.open({
	templateUrl: 'botActionTemplate',
	controller: botActionCtrl,
	resolve: {
	  clientId: () => row.clientId,
	  login: () => row.login,
	}
      })
    }

    $scope.showScanUrl = (row) => {
      $modal.open({
	templateUrl: 'scanUrlTemplate',
	controller: scanUrlCtrl,
	resolve: {
	  clientId: () => row.clientId,
	  login: () => row.login,
	  botId: () => row.botId,
	}
      })
    }
  }

  app.controller("qqLoginCtrl", qqLoginCtrl)
  qqLoginCtrl.$inject = ["$scope", "$uibModalInstance", "toastr", "buildModel", "buildPromise", "tools", "clientId", "botId"];
  function qqLoginCtrl($scope, $uibModalInstance, toastr, buildModel, buildPromise, tools, clientId, botId) {
    $scope.clientId = clientId;
    $scope.data = {};
    $scope.data.clientId = clientId;
    $scope.data.clientId = botId;
    
    $scope.close = function() {
      $uibModalInstance.dismiss();
    }

    $scope.login = function(data) {
      buildModel('botlogin', data).post((data) => {
	toastr.info(data, '信息')
      })

      $scope.close()
    }
  }

  app.controller("wechatLoginCtrl", wechatLoginCtrl)
  wechatLoginCtrl.$inject = ["$scope", "$uibModalInstance", "toastr", "buildModel", "buildPromise", "tools", "clientId", "botId"]
  function wechatLoginCtrl($scope, $uibModalInstance, toastr, buildModel, buildPromise, tools, clientId, botId) {
    $scope.clientId = clientId
    $scope.data = {}
    $scope.data.clientId = clientId
    $scope.data.botId = botId
    $scope.data.clientType = "WECHATBOT"
    
    $scope.close = () => {
      $uibModalInstance.dismiss()
    }

    $scope.login = (data) => {
      buildModel('botlogin', data).post((data) => {
	toastr.info(data, '信息')
      }, (error) => {
	toastr.error(error, '登录失败')
      })

      $scope.close()
    }
  }

  app.controller("botActionCtrl", botActionCtrl)
  botActionCtrl.$inject = ["$http", "$scope", "$uibModalInstance", "toastr", "buildModel", "buildModelResId", "buildPromise", "tools", "clientId", "login"]
  function botActionCtrl($http, $scope, $uibModalInstance, toastr, buildModel, buildModelResId, buildPromise, tools, clientId, login) {
    $scope.clientId = clientId
    $scope.data = {}
    $scope.data.clientId = clientId
    $scope.data.login = login

    const commonParams = {
      "toUserName" : {
        "name": "toUserName",
        "display": "收件人",
        "type": "string",
        "required": true,
        "comment": "收件人的wxid或者groupid",
      },
      "content" : {
        "name": "content",
        "display": "内容",
        "type": "string",
        "required": true,
      },
      "userId": {
        "name": "userId",
        "display": "联系人ID",
        "type": "string",
        "required": true,
        "comment": "联系人的wxid",
      },
      "stranger": {
        "name": "stranger",
        "display": "stranger",
        "type": "string",
        "required": true,
      },
      "ticket": {
        "name": "ticket",
        "display": "ticket",
        "type": "string",
        "required": true,
      },
      "type": {
        "name": "type",
        "display": "type",
        "type": "int",
        "required": true,
      },
      "verifyMessage": {
        "name": "content",
        "display": "content",
        "type": "string",
        "required": true,
        "comment": "验证信息",
      },
      "groupId": {
        "name": "groupId",
        "display": "群ID",
        "type": "string",
        "required": true,
      },
      "memberId": {
        "name": "memberId",
        "display": "群成员ID",
        "type": "string",
        "required": true,
      },
      "momentId": {
        "name": "momentId",
        "display": "动态ID",
        "type": "string",
        "required": true,
      },
      "momentIdOptional": {
        "name": "momentId",
        "display": "动态ID",
        "type": "string",
        "required": false,
      },
    }

    $scope.actions = {
      "SendTextMessage": [
        commonParams.toUserName,
        commonParams.content,
        {
          "name": "atList",
          "display": "点名列表",
          "type": "string",
          "required": false,
        }
      ],
      "SendAppMessage" : [
        commonParams.toUserName,
        {
          "name": "object",
          "display": "消息体",
          "type": "string",
          "required": true,
          "comment": "json结构的消息体",
        }
      ],
      "SearchContact": [
        commonParams.userId,
      ],
      "AddContact" : [
        commonParams.stranger,
        commonParams.ticket,
        
      ],
      "AcceptUser": [
        commonParams.stranger,
        commonParams.ticket,
        commonParams.verifyMessage,
      ],
      "SayHello": [
        commonParams.stranger,
        commonParams.ticket,
        commonParams.verifyMessage,
      ],
      "DeleteContact": [
        commonParams.userId,
      ],
      "CreateRoom": [
        {
          "name": "userList",
          "display": "用户列表",
          "type": "string",
          "required": true,
          "comment": "必须是好友才能加群",
        },
      ],
      "GetRoomMembers": [
        commonParams.groupId,
      ],
      "AddRoomMember": [
        commonParams.groupId,
        commonParams.memberId,
      ],
      "InviteRoomMember": [
        commonParams.groupId,
        commonParams.memberId,
      ],
      "DeleteRoomMember": [
        commonParams.groupId,
        commonParams.memberId,
      ],
      "SetRoomName": [
        commonParams.groupId,
        commonParams.content,
      ],
      "GetRoomQRCode": [
        commonParams.groupId,
      ],
      "SetRoomAnnouncement": [
        commonParams.groupId,
        commonParams.content,
      ],
      "SnsSendMoment": [
        commonParams.content,
      ],
      "SnsTimeline": [
        commonParams.momentIdOptional,
      ],
      "SnsUserPage": [
        commonParams.userId,
        commonParams.momentIdOptional,
      ],
      "SnsGetObject": [
        commonParams.momentId,
      ],
      "SnsComment": [
        commonParams.userId,
        commonParams.momentId,
        commonParams.content,
      ],
      "SnsLike": [
        commonParams.userId,
        commonParams.momentId,
      ],
      "SnsUpload": [
        {
          "name": "file",
          "display": "文件",
          "type": "string",
          "required": true,
          "comment":"上传图片的base64串，此接口仅上传，不会发送到朋友圈"
        }
      ],
      "SnsobjectOP": [
        commonParams.momentId,
        commonParams.type,
        {
          "name": "commentId",
          "display": "评论ID",
          "type": "string",
          "required": true,
        },
        {
          "name": "commentType",
          "display": "评论类型",
          "type": "int",
          "required": true,
        },
      ],      
    }

    $scope.close = () => {
      $uibModalInstance.dismiss();
    }

    let url = "/botaction/" + $scope.data.login
    
    $scope.sendAction = (data) => {
      $http({
	method: 'POST',
	url: url,
	data: JSON.stringify(data)
      })
	.then((success) => {
	  toastr.success(success, '发送成功')
	}, (error) => {
	  toastr.error(error, '发送失败')
	})

      $scope.close()
    }    
  }

  app.controller('editBotCtrl', editBotCtrl)
  editBotCtrl.$inject = ["$http", "$scope", "$uibModalInstance", "toastr", "buildModel", "buildModelResId", "buildPromise", "tools", "clientId", "clientType", "botName", "login", "filterId", "callback", "wxaappId"]
  function editBotCtrl($http, $scope, $uibModalInstance, toastr, buildModel, buildModelResId, buildPromise, tools, clientId, clientType, botName, login, filterId, callback, wxaappId) {
    $scope.data = {
      clientId: clientId,
      clientType: clientType,
      login: login,
      filterId: filterId,
      botName: botName,
      callback: callback,
      wxaappId: wxaappId,
    }

    $scope.close =  () => {
      $uibModalInstance.dismiss()
    }

    let url = "/bots/" + $scope.data.login

    $scope.saveBot = function(data) {
      $http({
	method: 'PUT',
	url: url,
	data: data
      }).then((success) => {
	toastr.success(success, '编辑成功')
      }, (error) => {
	toastr.error(error, '编辑失败')
      })

      $scope.close()
    }    
  }

  app.controller('createBotCtrl', createBotCtrl)
  createBotCtrl.$inject = ["$http", "$scope", "$uibModalInstance", "toastr", "buildModel", "buildModelResId", "buildPromise", "tools"]
  function createBotCtrl($http, $scope, $uibModalInstance, toastr, buildModel, buildModelResId, buildPromise, tools) {
    $scope.data = {}
    
    $scope.close = () => {
      $uibModalInstance.dismiss()
    }

    let url = "/bots"
    $scope.createBot = (data) => {
      $http({
	method: 'POST',
	url: url,
	data: data
      }).then((success) => {
	toastr.success($scope.data.botName, '创建成功')
      }, (error) => {
	toastr.error(error, '创建失败')
      })

      $scope.close()
    }
  }

  app.controller('scanWechatBotCtrl', scanWechatBotCtrl)
  scanWechatBotCtrl.$inject = ["$http", "$scope", "$uibModalInstance", "toastr", "buildModel", "buildModelResId", "buildPromise", "tools", "clientType"]
  function scanWechatBotCtrl($http, $scope, $uibModalInstance, toastr, buildModel, buildModelResId, buildPromise, tools, clientType) {
    $scope.data = {}
    $scope.data.clientType = clientType
    
    $scope.close = () => {
      $uibModalInstance.dismiss()
    }

    let url = "/bots/scancreate"
    $scope.scanCreateBot = (data) => {
      $http({
	method: 'POST',
	url: url,
	data: data,
      }).then((success) => {
	toastr.success($scope.data.botName, '扫码创建中')
      }, (error) => {
	toastr.error(error, '扫码创建失败')
      })
    }
    
    $scope.close()
  }

  app.controller('scanUrlCtrl', scanUrlCtrl)
  scanUrlCtrl.$inject = ["$http", "$scope", "$uibModalInstance", "toastr", "buildModel", "buildModelResId", "buildPromise", "tools", "botId"]
  function scanUrlCtrl($http, $scope, $uibModalInstance, toastr, buildModel, buildModelResId, buildPromise, tools, botId) {
    $scope.data = {}
    $scope.flag = true
    
    $scope.close = () => {
      $scope.flag = false
      $uibModalInstance.dismiss()
    }

    $scope.refresh = () => {
      buildPromise(buildModelResId('bots/id', $scope.botId)).then((data) => {
	if(data.body !== undefined) {
	  let bot = data.body
	  if(bot.scanUrl !== undefined) {
	    $scope.data.scanUrl = bot.scanUrl
	  } else {
	    if ($scope.flag) {
	      setTimeout(() => {
		$scope.refresh()
	      }, 3000)
	    }
	  }
	} else {
	  if ($scope.flag) {
	    setTimeout(() => {
	      $scope.refresh()
	    }, 3000)
	  }
	}
      })
    }
    
    $scope.botId = botId
    $scope.refresh()
  } 
})();




	  

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
          momentFilterId: () => row.momentFilterId,
	  callback: () => row.callback,
          wxaappId: () => row.wxaappId,
          botId: () => row.botId,
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

    $scope.clearLoginInfo = (row) => {
      buildPromise(buildModelResId('bots', row.botId + '/clearlogininfo').update((data) => {
        toastr.success(data, '清除成功')
      }))
    }

    $scope.shutdown = (row) => {
      buildPromise(buildModelResId('bots', row.botId + '/shutdown').update((data) => {
        toastr.success(data, '关闭成功')
      }))
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

    $scope.botLogout = (row) => {
      buildPromise(buildModelResId('bots', row.botId + "/logout").update((data) => {
        toastr.success(data, '登出成功')
      }))
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
      "fromUserName": {
        "name": "fromUserName",
        "display": "发件人",
        "type": "string",
        "required": true,
        "comment": "发件人的wxid",
      },
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
        "type": "longstring",
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
      "labelId": {
        "name": "labelId",
        "display": "标签ID",
        "type": "string",
        "required": true,
      }
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
          "type": "longstring",
          "required": false,
          "comment": "json结构的消息体",
        },
        {
          "name": "xml",
          "dispaly": "xml",
          "type": "longstring",
          "required": false,
          "comment": "xml结构的消息体，object和xml至少有一个必须",
        }
      ],
      "GetContact": [
        commonParams.userId,
      ],
      "SearchContact": [
        commonParams.userId,
      ],
      "AddContact" : [
        commonParams.stranger,
        commonParams.ticket,
        commonParams.type,
        {
          "name": "content",
          "display": "验证消息",
          "type": "string",
          "required": false,
        }
      ],
      "AcceptUser": [
        {
          "name": "EncryptUserName",
          "display": "EncryptUserName",
          "type": "string",
          "required": false,
        },
        commonParams.ticket,
        {
          "name": "content",
          "display": "content",
          "type": "string",
          "required": false,
          "comment": "验证信息",
        },
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
          "name": "memberList",
          "display": "用户列表",
          "type": "longstring",
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
          "type": "longstring",
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
      "GetLabelList":[],
      "AddLabel":[
        {
          "name": "label",
          "display": "label 标签",
          "type": "string",
          "required": true,
        },
      ],
      "DeleteLabel": [
        commonParams.labelId,
      ],
      "SetLabel": [
        {
          "name": "labelIdList",
          "display": "标签Id列表，逗号分隔",
          "type": "string",
          "required": true,
        },
        commonParams.userId,
      ],
      "GetRequestToken": [
        {
          "name": "ghName",
          "display": "公众号ID",
          "type": "string",
          "required": true,
        }, {
          "name": "url",
          "display": "url",
          "type": "string",
          "required": true,
        },
      ],
      "RequestUrl": [
        {
          "name": "url",
          "display": "url",
          "type": "string",
          "required": true,
        }, {
          "name": "xKey",
          "display": "xKey",
          "type": "string",
          "required": true,
        }, {
          "name": "xUin",
          "display": "xUin",
          "type": "string",
          "required": true,
        },
      ],
      "SyncContact": [],
    }
    

    $scope.close = () => {
      $uibModalInstance.dismiss();
    }

    let url = "/botaction/" + $scope.data.login
    
    $scope.sendAction = (data) => {
      if (!data.actionBody) {
        data.actionBody = JSON.stringify({})
      } else {
        // if(data.actionBody.object) {
        //   data.actionBody.object = JSON.parse(data.actionBody.object)
        // }
        
        if(data.actionBody.memberList) {
          data.actionBody.memberList = JSON.parse(data.actionBody.memberList)
        }

        if(data.actionBody.type) {
          data.actionBody.type = parseInt(data.actionBody.type, 10)
        }

        if(data.actionBody.commentType) {
          data.actionBody.commentType = parseInt(data.actionBody.commentType, 10)
        }

        data.actionBody = JSON.stringify(data.actionBody)
      }
      
      $http({
	method: 'POST',
	url: url,
	data: JSON.stringify(data),
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
  editBotCtrl.$inject = ["$http", "$scope", "$uibModalInstance", "toastr", "buildModel", "buildModelResId", "buildPromise", "tools", "clientId", "clientType", "botName", "login", "filterId", "momentFilterId", "callback", "wxaappId", "botId"]
  function editBotCtrl($http, $scope, $uibModalInstance, toastr, buildModel, buildModelResId, buildPromise, tools, clientId, clientType, botName, login, filterId, momentFilterId, callback, wxaappId, botId) {
    $scope.data = {
      clientId: clientId,
      clientType: clientType,
      login: login,
      filterId: filterId,
      momentFilterId: momentFilterId,
      botName: botName,
      callback: callback,
      wxaappId: wxaappId,
      botId: botId,
    }

    $scope.close =  () => {
      $uibModalInstance.dismiss()
    }

    let url = "/bots/" + $scope.data.botId

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

    $scope.rebuildMsgFilters = function(data) {
      $http({
        method: 'POST',
        url: '/bots/' + $scope.data.botId + '/msgfilters/rebuild'
      }).then((success) => {
        toastr.success(success, '重建成功')
      }, (error) => {
        toastr.error(error, '重建失败')
      })
    }

    $scope.rebuildMomentFilters = function(data) {
      $http({
        method: 'POST',
        url: '/bots/' + $scope.data.botId + '/momentfilters/rebuild'
      }).then((success) => {
        toastr.success(success, '重建成功')
      }, (error) => {
        toastr.error(error, '重建失败')
      })
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
      buildPromise(buildModelResId('bots', $scope.botId)).then((data) => {
	if(data.body !== undefined) {
	  let bot = data.body
	  if(bot.scanUrl !== undefined) {
	    $scope.data.scanUrl = bot.scanUrl
            $scope.flag = false
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




	  

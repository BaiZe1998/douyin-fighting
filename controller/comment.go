package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/goldenBill/douyin-fighting/service"
	"github.com/goldenBill/douyin-fighting/util"
	"net/http"
)

// CommentActionRequest 评论操作的请求
type CommentActionRequest struct {
	UserID      uint64 `form:"user_id" json:"user_id"`
	Token       string `form:"token" json:"token"`
	VideoID     uint64 `form:"video_id" json:"video_id"`
	ActionType  uint   `form:"action_type" json:"action_type"`
	CommentText string `form:"comment_text" json:"comment_text"`
	CommentID   uint64 `form:"comment_id" json:"comment_id"`
}

type CommentActionResponse struct {
	Response
	Comment Comment `json:"comment,omitempty"`
}

// CommentListRequest 评论列表的请求
type CommentListRequest struct {
	UserID  uint64 `form:"user_id" json:"user_id"`
	Token   string `form:"token" json:"token"`
	VideoID uint64 `form:"video_id" json:"video_id"`
}

// CommentListResponse 评论列表的响应
type CommentListResponse struct {
	Response
	CommentList []Comment `json:"comment_list,omitempty"`
}

// CommentAction no practical effect, just check if token is valid
func CommentAction(c *gin.Context) {
	// 参数绑定
	var r CommentActionRequest
	err := c.ShouldBind(&r)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{StatusCode: 1, StatusMsg: "bind error"})
		return
	}

	// 判断 action_type 是否正确
	if r.ActionType != 1 && r.ActionType != 2 {
		// action_type 不合法
		c.JSON(http.StatusBadRequest, Response{StatusCode: 1, StatusMsg: "action type error"})
		return
	}
	// 获取 userID
	r.UserID = c.GetUint64("UserID")

	// 判断 video_id 是否正确

	// 评论操作
	if r.ActionType == 1 {
		commentDao, err := service.AddComment(r.UserID, r.VideoID, r.CommentText)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{StatusCode: 1, StatusMsg: "comment failed"})
			return
		}
		userDao, _ := service.UserInfoByUserID(commentDao.UserID)
		isFollow, err := service.GetIsFollowStatus(r.UserID, userDao.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{StatusCode: 1, StatusMsg: err.Error()})
			return
		}
		c.JSON(http.StatusOK, CommentActionResponse{
			Response: Response{StatusCode: 0},
			Comment: Comment{
				ID: commentDao.CommentID,
				User: User{
					ID:             userDao.UserID,
					Name:           userDao.Name,
					FollowCount:    userDao.FollowCount,
					FollowerCount:  userDao.FollowerCount,
					TotalFavorited: userDao.TotalFavorited,
					FavoriteCount:  userDao.FavoriteCount,
					IsFollow:       isFollow,
				},
				Content:    commentDao.Content,
				CreateDate: commentDao.CreatedAt.Format("01-02"),
			},
		})
	} else {
		err = service.DeleteComment(r.UserID, r.VideoID, r.CommentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{StatusCode: 1, StatusMsg: "comment failed"})
			return
		}
		c.JSON(http.StatusOK, Response{StatusCode: 0})
	}
}

// CommentList all videos have same demo comment list
func CommentList(c *gin.Context) {
	// 参数绑定
	var r CommentListRequest
	err := c.ShouldBind(&r)
	if err != nil {
		c.JSON(http.StatusOK, Response{StatusCode: 1, StatusMsg: "bind error"})
		return
	}

	commentDaoList, userDaoList := service.GetCommentListAndUserList(r.VideoID)

	var (
		isFollowList []bool
		isLogged     = false // 用户是否传入了合法有效的token（是否登录）
		isFollow     bool
	)

	var userID uint64
	// 判断传入的token是否合法，用户是否存在
	if token := c.Query("token"); token != "" {
		claims, err := util.ParseToken(token)
		if err == nil {
			userID = claims.UserID
			if service.IsUserIDExist(userID) {
				isLogged = true
			}
		}
	}

	if isLogged {
		// 当用户登录时 一次性获取用户是否点赞了列表中的视频以及是否关注了视频的作者
		authorIDList := make([]uint64, len(commentDaoList))
		for i, user_ := range userDaoList {
			authorIDList[i] = user_.UserID
		}

		isFollowList, err = service.GetIsFollowStatusList(userID, authorIDList)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{StatusCode: 1, StatusMsg: err.Error()})
			return
		}
	}

	commentList := make([]Comment, 0, len(commentDaoList))
	for i := 0; i < len(commentDaoList); i++ {
		// 未登录时默认为未关注未点赞
		if isLogged {
			// 当用户登录时，判断是否关注当前作者
			isFollow = isFollowList[i]
		} else {
			isFollow = false
		}
		user := User{
			ID:             userDaoList[i].UserID,
			Name:           userDaoList[i].Name,
			FollowCount:    userDaoList[i].FollowCount,
			FollowerCount:  userDaoList[i].FollowerCount,
			TotalFavorited: userDaoList[i].TotalFavorited,
			FavoriteCount:  userDaoList[i].FavoriteCount,
			IsFollow:       isFollow,
		}
		comment := Comment{
			ID:         commentDaoList[i].CommentID,
			User:       user,
			Content:    commentDaoList[i].Content,
			CreateDate: commentDaoList[i].CreatedAt.Format("01-02"),
		}
		commentList = append(commentList, comment)
	}
	c.JSON(http.StatusOK, CommentListResponse{
		Response:    Response{StatusCode: 0},
		CommentList: commentList,
	})
}

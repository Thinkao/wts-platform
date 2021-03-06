package controller

import (
	uuid "github.com/satori/go.uuid"
	"server/account/constant"
	"server/account/model"
	"server/account/serializer"
	"server/account/utils"
	"server/account/validation"
	"server/setting/config"
	db "server/setting/model"
	"server/setting/request"
	"strconv"
)

type LoginAPI struct{ request.Controller }
type RegistAPI struct{ request.Controller }
type LogoutAPI struct{ request.Controller }
type UserAPI struct{ request.Controller }
type UserCountAPI struct {
	request.Controller
}
type DynamicAPI struct{ request.Controller }
type CommentsAPI struct{ request.Controller }
type HistotyAPI struct {
	request.Controller
}
type CSRFTokenAPI struct{ request.Controller }

func (c *LoginAPI) Post() {
	data := validation.LoginValid{}
	c.Check(&data, false)
	phoneEmail := data.PhoneEmail
	password := data.Password

	user := model.User{}
	if db.GetDB().Where("phone = ? or email = ?", phoneEmail, phoneEmail).First(&user).Error != nil {
		c.Error("账号不存在!")
		return
	}

	if user.Password != utils.Encrypt(password) {
		c.Error("密码错误!")
	}

	c.Login(user)
	c.Success(nil)
}

func (c *LogoutAPI) Post() {
	c.Check(nil, true, "all")
	c.Logout()
	c.Success(nil)
}

func (c *RegistAPI) Post() {
	data := validation.RegistValid{}
	c.Check(&data, false)
	phone := data.Phone
	password := data.Password
	username := data.Username

	if db.GetDB().Where("phone = ?", phone).First(&model.User{}).Error == nil {
		c.Error("该手机号已被注册!")
		return
	}

	if db.GetDB().Where("username = ?", username).First(&model.User{}).Error == nil {
		c.Error("该用户名已被注册!")
		return
	}

	user := model.User{Username: username, Password: utils.Encrypt(password), Phone: phone}
	db.GetDB().Create(&user)

	var user1 model.User
	if (db.GetDB().Table("user").Select("id").Where("phone = ?", phone).Scan(&user1)).Error == nil {
		id := user1.ID
		userInfo := model.UserInfo{UserID: id}
		db.GetDB().Create(&userInfo)
	}

	c.Success(nil)
}

func (c *UserCountAPI) Get() {
	if c.RequestUser().UserType != constant.Admin {
		c.Error("权限不足")
		return
	}
	var count = 0
	db.GetDB().Table("user").Count(&count)
	c.Success(count)
}

func (c *UserAPI) Get() {
	c.Check(nil, true, "all")
	id, _ := c.GetInt("ID")
	username := c.GetString("Username")
	phone := c.GetString("Phone")
	email := c.GetString("Email")
	userType := c.GetString("UserType")
	currentPage, _ := c.GetInt("CurrentPage")
	pageSize, _ := c.GetInt("PageSize")

	type User struct {
		serializer.UserSerialize
	}
	var users []User
	user := model.User{}

	if id != 0 && (c.RequestUser().UserType == constant.Normal) {
		if id == c.RequestUser().ID {
			db.GetDB().Where("id = ?", id).First(&user)
			db.GetDB().Model(&user).Related(&user.UserInfo)
			c.Success(user)
			return
		}
		c.Error("权限不足")
		return
	}

	var newIdStr = ""
	if id == 0 {
		newIdStr = ""
	} else {
		newIdStr = strconv.Itoa(id)
	}
	newId := "%" + newIdStr + "%"
	newUsername := "%" + username + "%"
	newPhone := "%" + phone + "%"
	newEmail := "%" + email + "%"
	newUserType := "%" + userType + "%"

	if c.RequestUser().UserType != constant.Admin {
		c.Error("权限不足")
		return
	}

	if pageSize == 0 && currentPage == 0 {
		db.GetDB().Joins("left join user_info on user.id = user_info.user_id").Where("user.id like ? and user.username like ? and user.phone like ? and user.email like ? and user.user_type like ?", newId, newUsername, newPhone, newEmail, newUserType).Find(&user)
		c.Success(user)
	}

	if pageSize > 0 && currentPage > 0 {
		db.GetDB().Joins("left join user_info on user.id = user_info.user_id").Where("user.id like ? and user.username like ? and user.phone like ? and user.email like ? and user.user_type like ?", newId, newUsername, newPhone, newEmail, newUserType).Limit(pageSize).Offset((currentPage - 1) * pageSize).Order("user.id").Find(&users)
		c.Success(users)
	}
}

func (c *UserAPI) Post() {
	data := validation.AddUserValid{}
	c.Check(&data, true, "admin")
	phone := data.Phone
	username := data.Username
	password := data.Password
	userType := data.UserType
	email := data.Email
	declaration := data.Declaration
	integral := data.Integral
	level := data.Level
	avatar := data.Avatar

	if db.GetDB().Where("phone = ?", phone).First(&model.User{}).Error == nil {
		c.Error("该手机号已被注册!")
		return
	}

	user := model.User{Phone: phone, Username: username, Password: utils.Encrypt(password), Email: email, UserType: userType}
	db.GetDB().Create(&user)

	var user1 model.User
	if (db.GetDB().Table("user").Select("id").Where("phone = ?", phone).Scan(&user1)).Error == nil {
		id := user1.ID
		userInfo := model.UserInfo{UserID: id, Declaration: declaration, Integral: integral, Level: level, Avatar: avatar}
		db.GetDB().Create(&userInfo)
	}

	c.Success(nil)

}

func (c *UserAPI) Put() {
	data := validation.UpdateUserValid{}
	c.Check(&data, true, "all")
	id := data.Id
	phone := data.Phone
	username := data.Username
	match := data.Match
	email := data.Email
	password := data.Password
	declaration := data.Declaration
	avatar := data.Avatar

	userData := map[string]interface{}{"Phone": phone, "Username": username, "Password": password, "Email": email, "Match":match}

	userInfoData := map[string]interface{}{"Declaration": declaration, "Avatar": avatar}

	UserType := c.RequestUser().UserType
	if UserType == constant.Admin {

		userData["UserType"] = data.UserType
		userInfoData["Integral"] = data.Integral
		userInfoData["Level"] = data.Level

	}

	user := model.User{}
	userInfo := model.UserInfo{}

	if db.GetDB().Where("id = ?", id).First(&model.User{}).Error == nil {

		db.GetDB().Where("id = ?", id).Model(&user).Updates(userData)
		db.GetDB().Where("user_id = ?", id).Model(&userInfo).Updates(userInfoData)

	} else {
		c.Error("系统没有该人员")
		return
	}

	c.Success(nil)
}

func (c *UserAPI) Delete() {
	data := validation.DeleteByIdValid{}
	c.Check(&data, true, "admin")
	id := data.Id
	if db.GetDB().Where("id = ?", id).First(&model.User{}).Error == nil {
		db.GetDB().Delete(model.User{}, "id = ?", id)
	} else {
		c.Error("系统没有该人员")
		return
	}
	c.Success(nil)
}

func (c *DynamicAPI) Get() {

	c.Check(nil, true, "all")
	id, _ := c.GetInt("id")
	type Dynamic struct {
		serializer.DynamicSerialize
	}

	var results []model.Result
	if id != 0 {
		db.GetDB().Table("user").Select("user.username,user_info.avatar,dynamic.content,dynamic.img_path,dynamic.create_time").Joins("left join user_info on user.id = user_info.user_id").Joins("right join dynamic on dynamic.user_id = user.id").Where("user.id = ?", id).Order("dynamic.create_time desc").Scan(&results)

	} else {
		db.GetDB().Table("user").Select("user.username,user_info.avatar,dynamic.content,dynamic.img_path,dynamic.create_time").Joins("left join user_info on user.id = user_info.user_id").Joins("right join dynamic on dynamic.user_id = user.id").Order("dynamic.create_time desc").Scan(&results)
	}

	for i := 0; i < len(results); i++ {
		results[i].ImgPath = config.IMAGE_FILE_PATH + results[i].ImgPath + ".png"
	}

	c.Success(&results)
}

func (c *DynamicAPI) Post() {

	data := validation.DynamicValid{}
	c.Check(&data, true, "all")

	userId := c.RequestUser().ID

	content := data.Content
	imgPath := data.ImgPath

	var img_file_path string

	f, _, _ := c.GetFile("file")
	if f != nil {
		u2 := uuid.NewV4()
		img_file_path = config.IMAGE_FILE_PATH + u2.String() + ".png"
		c.SaveToFile("file", img_file_path)
		img := model.Dynamic{
			ImgPath: u2.String(),
		}
		c.Success(img)
		return
	}

	if db.GetDB().Where("id = ?", userId).First(&model.User{}).Error != nil {
		c.Error("系统没有该人员")
		return
	}

	if content == "" && imgPath == "" {
		c.Success(nil)
		return
	}

	dynamic := model.Dynamic{UserID: userId, Content: content, ImgPath: imgPath}
	db.GetDB().Create(&dynamic)
	c.Success(nil)
}

func (c *DynamicAPI) Delete() {

	data := validation.DeleteByIdValid{}
	c.Check(&data, true, "all")

	id := data.Id

	if db.GetDB().Where("id = ?", id).First(&model.Dynamic{}).Error == nil {
		db.GetDB().Delete(model.Dynamic{}, "id = ?", id)
	} else {
		c.Error("系统没有该条动态")
		return
	}
	c.Success(nil)

}

func (c *CommentsAPI) GET() {
	id, _ := c.GetInt("id")
	dynamicId, _ := c.GetInt("dynamic_id")
	userId, _ := c.GetInt("comments_id")

	var users []serializer.CommentsSerialize

	if id != 0 {
		user := serializer.CommentsSerialize{}
		db.GetDB().Where("id = ?", id).First(&user, "User")
		c.Success(user)
		return
	} else if dynamicId != 0 {
		db.GetDB().Where("dynamic_id = ?", dynamicId).First(&users, "User")
	} else if userId != 0 {
		db.GetDB().Where("user_id = ?", userId).First(&users, "User")
	} else {
		db.GetDB().Find(&users, "User")

	}
	c.Success(&users)
}

func (c *CommentsAPI) POST() {
	data := validation.CommentsValid{}
	c.Check(&data, false)

	userId := data.UserId
	dynamicId := data.DynamicId
	commentsId := data.CommentsId
	content := data.Content
	imgPath := data.ImgPath

	if db.GetDB().Where("id = ?", userId).First(&model.User{}).Error != nil {
		c.Error("系统没有该人员")
		return
	}

	comment := model.Comment{UserId: userId, DynamicId: dynamicId, CommentsId: commentsId, Content: content, ImgPath: imgPath}
	db.GetDB().Create(&comment)
	c.Success(nil)

}

func (c *CSRFTokenAPI) Get() {
	c.Success(map[string]string{"Csrftoken": c.XSRFToken()})
}

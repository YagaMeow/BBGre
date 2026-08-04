package service

import (
	"bbgre/global"
	"bbgre/middleware"
	"bbgre/model"
	"time"

	"github.com/gin-gonic/gin"
)

type NoteRequest struct {
	Content string    `json:"content" example:"123"`
	Date    time.Time `json:"date" example:"2024-07-06T15:16:00.000Z"`
}

type UpdateRequest struct {
	NoteRequest
	ID     uint `json:"id" binding:"required"`
	Status uint `json:"status"`
}

type QueryRequest struct {
	Page  uint       `json:"page" example:"1"`
	Max   uint       `json:"max" example:"100"`
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// @Summary	创建笔记
// @Produce	json
// @Param		request	body		NoteRequest	true	"请求体"
// @Param		x-token	header		string		true	"鉴权"
// @Success	200		{object}	string		"成功"
// @Failure	400		{object}	string		"请求错误"
// @Failure	500		{object}	string		"内部错误"
// @Router		/api/notes/ [post]
func CreateNote(c *gin.Context) {
	// userID, _ = c.Get("userID")

	var input = NoteRequest{}

	if err := c.ShouldBindJSON(&input); err != nil {
		middleware.Error(c, 400, "Params Error", err.Error())
		return
	}

	note := model.CalendarNote{
		Content: input.Content,
		Date:    input.Date,
		Status:  0,
	}

	if err := global.DB.Create(&note).Error; err != nil {
		middleware.Error(c, 500, "Create article failed", err.Error())
		return
	}

	middleware.Success(c, gin.H{
		"content":    note.Content,
		"date":       note.Date.Format(time.RFC3339),
		"created_at": note.CreatedAt.Format(time.RFC3339),
		"status":     note.Status,
		"id":         note.ID,
	})

}

// @Summary	更新笔记
// @Produce	json
// @Param		request	body		UpdateRequest	true	"请求体"
// @Param		x-token	header		string			true	"token"
// @Success	200		{object}	string			"成功"
// @Failure	400		{object}	string			"请求错误"
// @Failure	500		{object}	string			"内部错误"
// @Router		/api/notes/update [post]
func UpdateNote(c *gin.Context) {
	var input = UpdateRequest{}
	if err := c.ShouldBindJSON(&input); err != nil {
		middleware.Error(c, 400, "Params error", err.Error())
		return
	}
	var note = model.CalendarNote{}
	if err := global.DB.Where("id = ?", input.ID).First(&note).Error; err != nil {
		middleware.Error(c, 404, "Note not found", err.Error())
		return
	}

	note.Content = input.Content
	note.Status = input.Status

	if err := global.DB.Save(note).Error; err != nil {
		middleware.Error(c, 500, "Update failed", err.Error())
		return
	}

	middleware.Success(c, gin.H{
		"content":    note.Content,
		"id":         note.ID,
		"date":       note.Date,
		"status":     note.Status,
		"created_at": note.CreatedAt,
	})
}

// @Summary	删除笔记
// @Produce	json
// @Param		request	body		UpdateRequest	true	"请求体"
// @Param		x-token	header		string			true	"token"
// @Success	200		{object}	string			"成功"
// @Failure	400		{object}	string			"请求错误"
// @Failure	500		{object}	string			"内部错误"
// @Router		/api/notes/update [delete]
func DeleteNote(c *gin.Context) {
	var input = UpdateRequest{}
	if err := c.ShouldBindJSON(&input); err != nil {
		middleware.Error(c, 400, "Params error.", err)
		return
	}

	var note = model.CalendarNote{}
	if err := global.DB.Where("id = ?", input.ID).First(&note).Error; err != nil {
		middleware.Error(c, 404, "Note not found.", err)
		return
	}

	println(note.ID, note.Content)
	if err := global.DB.Delete(&note).Error; err != nil {
		middleware.Error(c, 500, "Delete note failed.", err)
		return
	}

	middleware.Success(c, gin.H{
		"id": note.ID,
	})
}

// @Summary	获取笔记
// @Produce	json
// @Param		request	body		QueryRequest	false	"请求体"
// @Success	200		{object}	string			"成功"
// @Failure	400		{object}	string			"请求错误"
// @Failure	500		{object}	string			"内部错误"
// @Router		/api/notes [post]
func GetNoteList(c *gin.Context) {
	input := QueryRequest{}
	if err := c.ShouldBindJSON(&input); err != nil {
		middleware.Error(c, 400, "Params error.", err)
		return
	}

	var notes []model.CalendarNote
	db := global.DB.Model(&model.CalendarNote{})
	if input.Start != nil {
		db = db.Where("date >= ?", *input.Start)
	}
	if input.End != nil {
		db = db.Where("date <= ?", *input.End)
	}

	if err := db.Find(&notes).Error; err != nil {
		middleware.Error(c, 404, "Notes not found.", err)
		return
	}
	m := make(map[string][]model.CalendarNote)
	for _, note := range notes {
		formatDate := note.Date.Format("2006-01-02")
		m[formatDate] = append(m[formatDate], note)
	}

	middleware.Success(c, m)
}

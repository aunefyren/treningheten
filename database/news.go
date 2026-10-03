package database

import (
	"errors"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func GetNewsPosts() ([]models.News, error) {

	var newsPosts []models.News

	newsPostsRecords := Instance.Order("date desc").Where("news.enabled = ?", true).Find(&newsPosts)

	if newsPostsRecords.Error != nil {
		return []models.News{}, newsPostsRecords.Error
	} else if newsPostsRecords.RowsAffected == 0 {
		return []models.News{}, nil
	}

	if len(newsPosts) == 0 {
		newsPosts = []models.News{}
	}

	return newsPosts, nil

}

// GetNewsPostByNewsID returns the enabled news post, or nil when there is none.
func GetNewsPostByNewsID(newsID uuid.UUID) (*models.News, error) {
	var newsPost models.News

	newsPostRecords := Instance.Where("news.enabled = ?", true).Where("news.id = ?", newsID).Find(&newsPost)
	if newsPostRecords.Error != nil {
		return nil, newsPostRecords.Error
	} else if newsPostRecords.RowsAffected != 1 {
		return nil, nil
	}

	return &newsPost, nil
}

// Set news post to disabled
func DeleteNewsPost(newsID uuid.UUID) error {
	var news models.News
	newsRecords := Instance.Model(news).Where("news.ID= ?", newsID).Update("enabled", false)
	if newsRecords.Error != nil {
		return newsRecords.Error
	}
	if newsRecords.RowsAffected != 1 {
		return errors.New("Failed to delete news post in database.")
	}
	return nil
}

package articles

type ArticleSerializer struct {
	Article ArticleModel
}

type ArticleResponse struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Body        string   `json:"body"`
	Slug        string   `json:"slug"`
	TagList     []string `json:"tagList"`
}

func tagsToStrings(tags []TagModel) []string {
	result := make([]string, len(tags))
	for i, tag := range tags {
		result[i] = tag.Tag
	}
	return result
}

func (serializer ArticleSerializer) Response() ArticleResponse {
	return ArticleResponse{
		Title:       serializer.Article.Title,
		Description: serializer.Article.Description,
		Body:        serializer.Article.Body,
		Slug:        serializer.Article.Slug,
		TagList:     tagsToStrings(serializer.Article.Tags),
	}
}

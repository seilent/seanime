package anilist

import (
	"fmt"
	"github.com/goccy/go-json"
	"seanime/internal/util"
	"strconv"
)

func FetchBaseAnimeMap(ids []int) (ret map[int]*BaseAnime, err error) {

	query := fmt.Sprintf(CompoundBaseAnimeDocument, newCompoundQuery(ids))

	requestBody, err := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": nil,
	})
	if err != nil {
		return nil, err
	}

	data, err := customQuery(requestBody, util.NewLogger())
	if err != nil {
		return nil, err
	}

	var res map[string]*BaseAnime

	dataB, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(dataB, &res)
	if err != nil {
		return nil, err
	}

	ret = make(map[int]*BaseAnime)
	for k, v := range res {
		id, err := strconv.Atoi(k[1:])
		if err != nil {
			return nil, err
		}
		ret[id] = v
	}

	return ret, nil
}

// FetchCompleteAnimeMap fetches multiple complete anime in a single request to reduce API calls
func FetchCompleteAnimeMap(ids []int) (ret map[int]*CompleteAnime, err error) {
	if len(ids) == 0 {
		return make(map[int]*CompleteAnime), nil
	}

	// Split into batches of 25 to avoid hitting query complexity limits
	const batchSize = 25
	ret = make(map[int]*CompleteAnime)

	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		
		batch := ids[i:end]
		query := fmt.Sprintf(CompoundCompleteAnimeDocument, newCompoundQuery(batch))

		requestBody, err := json.Marshal(map[string]interface{}{
			"query":     query,
			"variables": nil,
		})
		if err != nil {
			return nil, err
		}

		data, err := customQuery(requestBody, util.NewLogger())
		if err != nil {
			return nil, fmt.Errorf("batch %d-%d failed: %w", i, end-1, err)
		}

		var res map[string]*CompleteAnime
		dataB, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(dataB, &res)
		if err != nil {
			return nil, err
		}

		for k, v := range res {
			id, err := strconv.Atoi(k[1:])
			if err != nil {
				return nil, err
			}
			ret[id] = v
		}
	}

	return ret, nil
}

func newCompoundQuery(ids []int) string {
	var query string
	for _, id := range ids {
		query += fmt.Sprintf(`
		t%d: Media(id: %d) {
			...baseAnime
		}
		`, id, id)
	}
	return query
}

const CompoundBaseAnimeDocument = `query CompoundQueryTest {
%s
}
fragment baseAnime on Media {
	id
	idMal
	siteUrl
	status(version: 2)
	season
	type
	format
	bannerImage
	episodes
	synonyms
	isAdult
	countryOfOrigin
	meanScore
	description
	genres
	duration
	trailer {
		id
		site
		thumbnail
	}
	title {
		userPreferred
		romaji
		english
		native
	}
	coverImage {
		extraLarge
		large
		medium
		color
	}
	startDate {
		year
		month
		day
	}
	endDate {
		year
		month
		day
	}
	nextAiringEpisode {
		airingAt
		timeUntilAiring
		episode
	}
}`
const CompoundCompleteAnimeDocument = `query CompoundCompleteAnime {
%s
}
fragment completeAnime on Media {
	id
	idMal
	siteUrl
	status(version: 2)
	season
	seasonYear
	type
	format
	bannerImage
	episodes
	synonyms
	isAdult
	countryOfOrigin
	meanScore
	description
	genres
	tags {
		id
		name
		category
		rank
		isGeneralSpoiler
		isMediaSpoiler
		isAdult
	}
	duration
	trailer {
		id
		site
		thumbnail
	}
	title {
		userPreferred
		romaji
		english
		native
	}
	coverImage {
		extraLarge
		large
		medium
		color
	}
	startDate {
		year
		month
		day
	}
	endDate {
		year
		month
		day
	}
	nextAiringEpisode {
		airingAt
		timeUntilAiring
		episode
	}
	relations {
		edges {
			relationType(version: 2)
			node {
				...baseAnime
			}
		}
	}
}
fragment baseAnime on Media {
	id
	idMal
	siteUrl
	status(version: 2)
	season
	type
	format
	bannerImage
	episodes
	synonyms
	isAdult
	countryOfOrigin
	meanScore
	description
	genres
	duration
	trailer {
		id
		site
		thumbnail
	}
	title {
		userPreferred
		romaji
		english
		native
	}
	coverImage {
		extraLarge
		large
		medium
		color
	}
	startDate {
		year
		month
		day
	}
	endDate {
		year
		month
		day
	}
	nextAiringEpisode {
		airingAt
		timeUntilAiring
		episode
	}
}`

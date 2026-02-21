package cmd

import (
	"errors"
	"fmt"
	"iwaradl/api"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	// Sort strategy for list query.
	sort = "trending"

	// Number of pages to fetch from the API.
	pageLimit = 1

	// Only keep videos created within this many days.
	dateLimit = 7

	// Rating scope for queried videos.
	rating = "ecchi"

	// Base minimum likes for a video.
	filterLike0 = 100

	// Additional likes required per day since creation.
	filterLikeInc = 50

	// Minimum required views.
	filterViews = 0

	// Minimum required duration in seconds.
	filterDuration = 90
)

var outputListFile = "videolist.txt"

var validSortValues = map[string]struct{}{
	"date":       {},
	"trending":   {},
	"popularity": {},
	"views":      {},
	"likes":      {},
}

var validRatingValues = map[string]struct{}{
	"all":     {},
	"general": {},
	"ecchi":   {},
}

func IsAcceptVideo(v api.VideoInfo) bool {
	like := v.NumLikes
	view := v.NumViews
	dur := v.File.Duration
	createAt := v.CreatedAt

	t := int(time.Now().Sub(createAt).Hours() / 24)
	likeFilter := filterLike0 + filterLikeInc*t

	dateLimitTime := time.Now().AddDate(0, 0, -dateLimit)

	return like >= likeFilter && view >= filterViews && dur >= filterDuration && createAt.After(dateLimitTime)
}

func genVideoList() error {
	now := time.Now()
	dateLimitTime := now.AddDate(0, 0, -dateLimit)

	var videolist []api.VideoInfo

	// Pull video pages from API.
	for page := 0; page < pageLimit; page++ {
		if (page+1)%5 == 0 {
			time.Sleep(time.Minute)
		} else {
			time.Sleep(10 * time.Second)
		}
		fmt.Print("Getting page: ", page)
		videos, err := api.GetVideoList(sort, page, rating)
		if err != nil {
			return err
		}
		if len(videos.Results) == 0 {
			break
		}
		fmt.Println(" done. Got ", len(videos.Results), " videos")

		videolist = append(videolist, videos.Results...)

		if videos.Results[len(videos.Results)-1].CreatedAt.Before(dateLimitTime) && sort == "date" {
			break
		}
	}

	// Filter by engagement + freshness rules.
	var filteredVideolist []api.VideoInfo

	// Display creation time in local preference.
	loc, _ := time.LoadLocation("Asia/Shanghai")
	fmt.Println("      ID      \tLikes\t         Date         \t   Title")
	for _, video := range videolist {
		if IsAcceptVideo(video) {
			filteredVideolist = append(filteredVideolist, video)
			fmt.Printf("%s\t%5d\t%s \t%s\n", video.Id, video.NumLikes, video.CreatedAt.In(loc).Format("2006-01-02 15:04:05"), video.Title)
		}
	}

	// Write filtered video URLs to output file.
	f, err := os.Create(outputListFile)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	for _, video := range filteredVideolist {
		_, err := f.WriteString("https://www.iwara.tv/video/" + video.Id + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

func validateGenListParams() error {
	if _, ok := validSortValues[sort]; !ok {
		return fmt.Errorf("invalid --sort %q, allowed values: date, trending, popularity, views, likes", sort)
	}

	if _, ok := validRatingValues[rating]; !ok {
		return fmt.Errorf("invalid --rating %q, allowed values: all, general, ecchi", rating)
	}

	if pageLimit <= 0 {
		return fmt.Errorf("invalid --page-limit %d, must be greater than 0", pageLimit)
	}

	if dateLimit <= 0 {
		return fmt.Errorf("invalid --date-limit %d, must be greater than 0", dateLimit)
	}

	if filterLike0 < 0 {
		return fmt.Errorf("invalid --filter-like0 %d, must be greater than or equal to 0", filterLike0)
	}

	if filterLikeInc < 0 {
		return fmt.Errorf("invalid --filter-like-inc %d, must be greater than or equal to 0", filterLikeInc)
	}

	if filterViews < 0 {
		return fmt.Errorf("invalid --filter-views %d, must be greater than or equal to 0", filterViews)
	}

	if filterDuration <= 0 {
		return fmt.Errorf("invalid --filter-duration %d, must be greater than 0", filterDuration)
	}

	if strings.TrimSpace(outputListFile) == "" {
		return errors.New("invalid --output, file name cannot be empty")
	}

	return nil
}

var genListCmd = &cobra.Command{
	Use:   "genlist",
	Short: "Generate a filtered Iwara video URL list",
	Long:  "Query videos from Iwara by sort/rating, apply local filtering rules, and write the resulting video URLs to a text file.",
	Example: "  iwaradl genlist --sort date --page-limit 3 --date-limit 14 --output videolist.txt\n" +
		"  iwaradl genlist --rating all --filter-like0 200 --filter-like-inc 20 --filter-duration 120",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		initRuntimeConfig()

		if err := validateGenListParams(); err != nil {
			return err
		}

		return genVideoList()
	},
}

func init() {
	rootCmd.AddCommand(genListCmd)
	genListCmd.Flags().StringVar(&sort, "sort", "trending", "Sort strategy for list query. Allowed: date, trending, popularity, views, likes")
	genListCmd.Flags().IntVar(&pageLimit, "page-limit", 1, "Number of list pages to fetch (must be > 0)")
	genListCmd.Flags().IntVar(&dateLimit, "date-limit", 7, "Only keep videos created in the last N days (must be > 0)")
	genListCmd.Flags().StringVar(&rating, "rating", "ecchi", "Rating scope of list query. Allowed: all, general, ecchi")
	genListCmd.Flags().IntVar(&filterLike0, "filter-like0", 100, "Base minimum likes for a video (>= 0)")
	genListCmd.Flags().IntVar(&filterLikeInc, "filter-like-inc", 50, "Extra required likes per day since creation (>= 0)")
	genListCmd.Flags().IntVar(&filterViews, "filter-views", 0, "Minimum views for each video (>= 0)")
	genListCmd.Flags().IntVar(&filterDuration, "filter-duration", 90, "Minimum duration in seconds for each video (> 0)")
	genListCmd.Flags().StringVar(&outputListFile, "output", "videolist.txt", "Output file path for generated video URLs")
}

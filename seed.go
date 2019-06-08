package seed

import (
	"context"
	"crypto/sha1"
	"fmt"
	"strings"
	"sync"

	shell "github.com/godcong/go-ipfs-restapi"
	jsoniter "github.com/json-iterator/go"
	"github.com/yinhevr/seed/model"
)

type Options func(*Seed)

type Thread struct {
	wg sync.WaitGroup
}

type Threader interface {
}

type Runnable interface {
	Run(context.Context)
}

type Stepper int

const (
	StepperNone Stepper = iota
	StepperProcess
	StepperMove
	StepperJson
	StepperPin
	StepperMax
)

type Seeder interface {
	Start()
	Wait()
	Stop()
	Err() error
}

type Seed struct {
	Unfinished  []*model.Unfinished
	wg          *sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	shell       *shell.Shell
	threads     int
	runner      []Runnable
	ignores     map[string][]byte
	workspace   string
	processPath string
	err         error
}

func (seed *Seed) Stop() {
	if seed.cancel != nil {
		seed.cancel()
	}
}

func (seed *Seed) Err() error {
	return seed.err
}

func (seed *Seed) Start() {
	seed.wg.Add(1)
	go func() {
		log.Info("first running")
		defer seed.wg.Done()
		for i, r := range seed.runner {
			log.With("index", i)
			if r != nil {
				r.Run(seed.ctx)
			}
		}
	}()
}

func (seed *Seed) Wait() {
	seed.wg.Wait()
}

func NewSeeder(ops ...Options) Seeder {
	ctx, cancel := context.WithCancel(context.Background())
	seed := &Seed{
		wg:      &sync.WaitGroup{},
		ctx:     ctx,
		cancel:  cancel,
		threads: 0,
		runner:  make([]Runnable, StepperMax),
		ignores: make(map[string][]byte),
	}
	for _, op := range ops {
		op(seed)
	}

	if seed.shell == nil {
		seed.shell = shell.NewShell("localhost:5001")
	}

	return seed
}

func ShellOption(s *shell.Shell) Options {
	return func(seed *Seed) {
		seed.shell = s
	}
}

func ProcessOption(process *Process) Options {
	return func(seed *Seed) {
		process.seed = seed
		seed.runner[StepperProcess] = process
	}
}

func PinOption(pin *Pin) Options {
	return func(seed *Seed) {
		pin.seed = seed
		seed.runner[StepperPin] = pin
	}
}

func IgnoreOption(ignores ...string) Options {
	return func(seed *Seed) {
		for _, i := range ignores {
			seed.ignores[PathMD5(i)] = nil
		}
	}
}

func ThreadOption(t int) Options {
	return func(seed *Seed) {
		seed.threads = t
	}
}

// Extend ...
type Extend struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// VideoSource ...
type VideoSource struct {
	Bangumi      string    `json:"bangumi"`       //番号 no
	SourceHash   string    `json:"source_hash"`   //原片hash
	Type         string    `json:"type"`          //类型：film，FanDrama
	Output       string    `json:"output"`        //输出：3D，2D
	VR           string    `json:"vr"`            //VR格式：左右，上下，平面
	Thumb        string    `json:"thumb"`         //缩略图
	Intro        string    `json:"intro"`         //简介 title
	Alias        []string  `json:"alias"`         //别名，片名
	VideoEncode  string    `json:"video_encode"`  //视频编码
	AudioEncode  string    `json:"audio_encode"`  //音频编码
	Files        []string  `json:"files"`         //存放路径
	HashFiles    []string  `json:"hash_files"`    //已上传Hash
	CheckFiles   []string  `json:"check_files"`   //Unfinished checksum
	Slice        bool      `json:"slice"`         //是否HLS切片
	Encrypt      bool      `json:"encrypt"`       //加密
	Key          string    `json:"key"`           //秘钥
	M3U8         string    `json:"m3u8"`          //M3U8名
	SegmentFile  string    `json:"segment_file"`  //ts切片名
	PosterPath   string    `json:"poster_path"`   //海报路径
	Poster       string    `json:"poster"`        //海报HASH
	ExtendList   []*Extend `json:"extend_list"`   //扩展信息
	Role         []string  `json:"role"`          //角色列表 stars
	Director     []string  `json:"director"`      //导演
	Systematics  string    `json:"systematics"`   //分级
	Season       string    `json:"season"`        //季
	Episode      string    `json:"episode"`       //集数
	TotalEpisode string    `json:"total_episode"` //总集数
	Sharpness    string    `json:"sharpness"`     //清晰度
	Publish      string    `json:"publish"`       //发行日
	Date         string    `json:"date"`          //发行日
	Length       string    `json:"length"`        //片长
	Producer     string    `json:"producer"`      //制片商
	Series       string    `json:"series"`        //系列
	Tags         []string  `json:"tags"`          //标签
	Publisher    string    `json:"publisher"`     //发行商
	Language     string    `json:"language"`      //语言
	Caption      string    `json:"caption"`       //字幕
	MagnetLinks  []string  `json:"magnet_links"`  //磁链
	Uncensored   bool      `json:"uncensored"`    //有码,无码
}

// VideoList ...
var VideoList = LoadVideo()

// LoadVideo ...
func LoadVideo() []*model.Video {
	var videos []*model.Video
	e := ReadJSON("video.json", &videos)
	if e != nil {
		return nil
	}
	return videos
}

// ListVideoGet ...
func ListVideoGet(source *VideoSource) *model.Video {
	for _, v := range VideoList {
		if v.Bangumi == source.Bangumi {
			return v
		}
	}
	return NewVideo(source)
}

// VideoListAdd ...
func VideoListAdd(source *VideoSource, video *model.Video) {
	for i, v := range VideoList {
		if v.Bangumi == source.Bangumi {
			VideoList[i] = video
			return
		}
	}
	VideoList = append(VideoList, video)
}

// SaveVideos ...
func SaveVideos() (e error) {
	e = WriteJSON("video.json", VideoList)
	if e != nil {
		return e
	}
	return nil
}

func parseVideoBase(video *model.Video, source *VideoSource) {
	if video == nil {
		return
	}
	//always not null
	alias := []string{}
	aliasS := ""
	if source.Alias != nil || len(source.Alias) > 0 {
		alias = source.Alias
		aliasS = alias[0]
	}
	//always not null
	role := []string{}
	roleS := ""
	if source.Role != nil || len(source.Role) > 0 {
		role = source.Role
		roleS = role[0]
	}

	//always not null
	director := []string{}
	if source.Director != nil {
		director = source.Director
	}

	intro := source.Intro
	if intro == "" {
		intro = aliasS + " " + roleS
	}

	video.Bangumi = source.Bangumi
	//video.Type = source.Type
	//video.Format = source.Format
	//video.VR = source.VR
	video.Thumb = source.Thumb
	video.Intro = intro
	video.Alias = alias
	video.Role = role
	video.Director = director
	//video.Language = source.Language
	//video.Caption = source.Caption
	video.SourceHash = source.SourceHash
	video.Season = source.Season
	video.Episode = source.Episode
	video.TotalEpisode = source.TotalEpisode
	video.Publish = source.Publish

}

// NewVideo ...
func NewVideo(source *VideoSource) *model.Video {
	alias := []string{}
	if source.Alias != nil {
		alias = source.Alias
	}
	return &model.Video{
		Bangumi: strings.ToUpper(source.Bangumi),
		//Type:         source.Type,
		//Format:       source.Format,
		//VR:           source.VR,
		Thumb: source.Thumb,
		Intro: source.Intro,
		Alias: alias,
		//Language:     source.Language,
		//Caption:      source.Caption,
		Role:         source.Role,
		Director:     source.Director,
		Season:       source.Season,
		Episode:      source.Episode,
		TotalEpisode: source.TotalEpisode,
		Publish:      source.Publish,
		//VideoGroupList: nil,
		//SourceInfoList: nil,
	}
}

// Hash ...
func Hash(v interface{}) string {
	bytes, e := jsoniter.Marshal(v)
	if e != nil {
		return ""
	}
	return fmt.Sprintf("%x", sha1.Sum([]byte(bytes)))
}

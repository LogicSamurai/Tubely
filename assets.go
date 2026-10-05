package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

type ProbeResponse struct {
	Streams []ProbeStream `json:"streams"`
}

type ProbeStream struct {
	Index              int                `json:"index"`
	CodecName          *string            `json:"codec_name,omitempty"`
	CodecLongName      *string            `json:"codec_long_name,omitempty"`
	Profile            *string            `json:"profile,omitempty"`
	CodecType          string             `json:"codec_type"`
	CodecTagString     *string            `json:"codec_tag_string,omitempty"`
	CodecTag           *string            `json:"codec_tag,omitempty"`
	Width              *int               `json:"width,omitempty"`
	Height             *int               `json:"height,omitempty"`
	CodedWidth         *int               `json:"coded_width,omitempty"`
	CodedHeight        *int               `json:"coded_height,omitempty"`
	ClosedCaptions     *int               `json:"closed_captions,omitempty"`
	FilmGrain          *int               `json:"film_grain,omitempty"`
	HasBFrames         *int               `json:"has_b_frames,omitempty"`
	SampleAspectRatio  *string            `json:"sample_aspect_ratio,omitempty"`
	DisplayAspectRatio *string            `json:"display_aspect_ratio,omitempty"`
	PixelFormat        *string            `json:"pix_fmt,omitempty"`
	Level              *int               `json:"level,omitempty"`
	ColorRange         *string            `json:"color_range,omitempty"`
	ColorSpace         *string            `json:"color_space,omitempty"`
	ColorTransfer      *string            `json:"color_transfer,omitempty"`
	ColorPrimaries     *string            `json:"color_primaries,omitempty"`
	ChromaLocation     *string            `json:"chroma_location,omitempty"`
	FieldOrder         *string            `json:"field_order,omitempty"`
	Refs               *int               `json:"refs,omitempty"`
	IsAVC              *string            `json:"is_avc,omitempty"`
	NALLengthSize      *string            `json:"nal_length_size,omitempty"`
	ID                 *string            `json:"id,omitempty"`
	FrameRate          *string            `json:"r_frame_rate,omitempty"`
	AvgFrameRate       *string            `json:"avg_frame_rate,omitempty"`
	TimeBase           *string            `json:"time_base,omitempty"`
	StartPTS           *int64             `json:"start_pts,omitempty"`
	StartTime          *string            `json:"start_time,omitempty"`
	DurationTS         *int64             `json:"duration_ts,omitempty"`
	Duration           *string            `json:"duration,omitempty"`
	BitRate            *string            `json:"bit_rate,omitempty"`
	BitsPerRawSample   *string            `json:"bits_per_raw_sample,omitempty"`
	Frames             *string            `json:"nb_frames,omitempty"`
	SampleFormat       *string            `json:"sample_fmt,omitempty"`
	SampleRate         *string            `json:"sample_rate,omitempty"`
	Channels           *int               `json:"channels,omitempty"`
	ChannelLayout      *string            `json:"channel_layout,omitempty"`
	BitsPerSample      *int               `json:"bits_per_sample,omitempty"`
	InitialPadding     *int               `json:"initial_padding,omitempty"`
	ExtraDataSize      *int               `json:"extradata_size,omitempty"`
	Disposition        *StreamDisposition `json:"disposition,omitempty"`
	Tags               *StreamTags        `json:"tags,omitempty"`
}

type StreamTags struct {
	Language    *string `json:"language,omitempty"`
	HandlerName *string `json:"handler_name,omitempty"`
	VendorID    *string `json:"vendor_id,omitempty"`
	Encoder     *string `json:"encoder,omitempty"`
	Timecode    *string `json:"timecode,omitempty"`
}

type StreamDisposition struct {
	Default         *int `json:"default,omitempty"`
	Dub             *int `json:"dub,omitempty"`
	Original        *int `json:"original,omitempty"`
	Comment         *int `json:"comment,omitempty"`
	Lyrics          *int `json:"lyrics,omitempty"`
	Karaoke         *int `json:"karaoke,omitempty"`
	Forced          *int `json:"forced,omitempty"`
	HearingImpaired *int `json:"hearing_impaired,omitempty"`
	VisualImpaired  *int `json:"visual_impaired,omitempty"`
	CleanEffects    *int `json:"clean_effects,omitempty"`
	AttachedPic     *int `json:"attached_pic,omitempty"`
	TimedThumbnails *int `json:"timed_thumbnails,omitempty"`
	NonDiegetic     *int `json:"non_diegetic,omitempty"`
	Captions        *int `json:"captions,omitempty"`
	Descriptions    *int `json:"descriptions,omitempty"`
	Metadata        *int `json:"metadata,omitempty"`
	Dependent       *int `json:"dependent,omitempty"`
	StillImage      *int `json:"still_image,omitempty"`
}

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getVideoAspectRatio(filePath string) (string, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", err
	}

	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var output bytes.Buffer
	cmd.Stdout = &output

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	var probeResponse ProbeResponse
	if err := json.Unmarshal(output.Bytes(), &probeResponse); err != nil {
		return "", err
	}

	if len(probeResponse.Streams) == 0 || probeResponse.Streams[0].Width == nil || probeResponse.Streams[0].Height == nil {
		return "", fmt.Errorf("video stream width or height is missing")
	}

	if displayAspectRatio := probeResponse.Streams[0].DisplayAspectRatio; displayAspectRatio != nil {
		switch *displayAspectRatio {
		case "9:16":
			return "9:16", nil
		case "16:9":
			return "16:9", nil
		}
	}
	
	return "other", nil
}

func processVideoForFastStart(filePath string) (string, error) {
	newPath := filePath+".processing"
	cmd := exec.Command("ffmpeg","-i",filePath,"-c","copy","-movflags","faststart","-f","mp4",newPath)
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return newPath, nil
}


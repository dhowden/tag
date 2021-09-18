// Copyright 2015, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var atomTypes = map[int]string{
	0:  "implicit", // automatic based on atom name
	1:  "text",
	13: "jpeg",
	14: "png",
	21: "uint8",
}

// NB: atoms does not include "----", this is handled separately
var atoms = atomNames(map[string]string{
	"\xa9alb": "album",
	"\xa9art": "artist",
	"\xa9ART": "artist",
	"aART":    "album_artist",
	"\xa9day": "year",
	"\xa9nam": "title",
	"\xa9gen": "genre",
	"gnre":    "genre ID3v1 ID",
	"geID":    "genre ID",
	"trkn":    "track",
	"\xa9wrt": "composer",
	"\xa9too": "encoder",
	"cprt":    "copyright",
	"covr":    "picture",
	"\xa9grp": "grouping",
	"keyw":    "keyword",
	"\xa9lyr": "lyrics",
	"\xa9cmt": "comment",
	"tmpo":    "tempo",
	"cpil":    "compilation",
	"disk":    "disc",
})

var genreIDValues = map[int]string{
	2:        "Blues",
	3:        "Comedy",
	4:        "Children's Music",
	5:        "Classical",
	6:        "Country",
	7:        "Electronic",
	8:        "Holiday",
	9:        "Classical|Opera",
	10:       "Singer/Songwriter",
	11:       "Jazz",
	12:       "Latino",
	13:       "New Age",
	14:       "Pop",
	15:       "R&B/Soul",
	16:       "Soundtrack",
	17:       "Dance",
	18:       "Hip-Hop/Rap",
	19:       "World",
	20:       "Alternative",
	21:       "Rock",
	22:       "Christian & Gospel",
	23:       "Vocal",
	24:       "Reggae",
	25:       "Easy Listening",
	27:       "J-Pop",
	28:       "Enka",
	29:       "Anime",
	30:       "Kayokyoku",
	50:       "Fitness & Workout",
	51:       "Pop|K-Pop",
	52:       "Karaoke",
	53:       "Instrumental",
	1001:     "Alternative|College Rock",
	1002:     "Alternative|Goth Rock",
	1003:     "Alternative|Grunge",
	1004:     "Alternative|Indie Rock",
	1005:     "Alternative|New Wave",
	1006:     "Alternative|Punk",
	1007:     "Blues|Chicago Blues",
	1009:     "Blues|Classic Blues",
	1010:     "Blues|Contemporary Blues",
	1011:     "Blues|Country Blues",
	1012:     "Blues|Delta Blues",
	1013:     "Blues|Electric Blues",
	1014:     "Children's Music|Lullabies",
	1015:     "Children's Music|Sing-Along",
	1016:     "Children's Music|Stories",
	1017:     "Classical|Avant-Garde",
	1018:     "Classical|Baroque Era",
	1019:     "Classical|Chamber Music",
	1020:     "Classical|Chant",
	1021:     "Classical|Choral",
	1022:     "Classical|Classical Crossover",
	1023:     "Classical|Early Music",
	1024:     "Classical|Impressionist",
	1025:     "Classical|Medieval Era",
	1026:     "Classical|Minimalism",
	1027:     "Classical|Modern Era",
	1028:     "Classical|Opera",
	1029:     "Classical|Orchestral",
	1030:     "Classical|Renaissance",
	1031:     "Classical|Romantic Era",
	1032:     "Classical|Wedding Music",
	1033:     "Country|Alternative Country",
	1034:     "Country|Americana",
	1035:     "Country|Bluegrass",
	1036:     "Country|Contemporary Bluegrass",
	1037:     "Country|Contemporary Country",
	1038:     "Country|Country Gospel",
	1039:     "Country|Honky Tonk",
	1040:     "Country|Outlaw Country",
	1041:     "Country|Traditional Bluegrass",
	1042:     "Country|Traditional Country",
	1043:     "Country|Urban Cowboy",
	1044:     "Dance|Breakbeat",
	1045:     "Dance|Exercise",
	1046:     "Dance|Garage",
	1047:     "Dance|Hardcore",
	1048:     "Dance|House",
	1049:     "Dance|Jungle/Drum'n'bass",
	1050:     "Dance|Techno",
	1051:     "Dance|Trance",
	1052:     "Jazz|Big Band",
	1053:     "Jazz|Bop",
	1054:     "Easy Listening|Lounge",
	1055:     "Easy Listening|Swing",
	1056:     "Electronic|Ambient",
	1057:     "Electronic|Downtempo",
	1058:     "Electronic|Electronica",
	1060:     "Electronic|IDM/Experimental",
	1061:     "Electronic|Industrial",
	1062:     "Singer/Songwriter|Alternative Folk",
	1063:     "Singer/Songwriter|Contemporary Folk",
	1064:     "Singer/Songwriter|Contemporary Singer/Songwriter",
	1065:     "Singer/Songwriter|Folk-Rock",
	1066:     "Singer/Songwriter|New Acoustic",
	1067:     "Singer/Songwriter|Traditional Folk",
	1068:     "Hip-Hop/Rap|Alternative Rap",
	1069:     "Hip-Hop/Rap|Dirty South",
	1070:     "Hip-Hop/Rap|East Coast Rap",
	1071:     "Hip-Hop/Rap|Gangsta Rap",
	1072:     "Hip-Hop/Rap|Hardcore Rap",
	1073:     "Hip-Hop/Rap|Hip-Hop",
	1074:     "Hip-Hop/Rap|Latin Rap",
	1075:     "Hip-Hop/Rap|Old School Rap",
	1076:     "Hip-Hop/Rap|Rap",
	1077:     "Hip-Hop/Rap|Underground Rap",
	1078:     "Hip-Hop/Rap|West Coast Rap",
	1079:     "Holiday|Chanukah",
	1080:     "Holiday|Christmas",
	1081:     "Holiday|Christmas: Children's",
	1082:     "Holiday|Christmas: Classic",
	1083:     "Holiday|Christmas: Classical",
	1084:     "Holiday|Christmas: Jazz",
	1085:     "Holiday|Christmas: Modern",
	1086:     "Holiday|Christmas: Pop",
	1087:     "Holiday|Christmas: R&B",
	1088:     "Holiday|Christmas: Religious",
	1089:     "Holiday|Christmas: Rock",
	1090:     "Holiday|Easter",
	1091:     "Holiday|Halloween",
	1092:     "Holiday|Holiday: Other",
	1093:     "Holiday|Thanksgiving",
	1094:     "Christian & Gospel|CCM",
	1095:     "Christian & Gospel|Christian Metal",
	1096:     "Christian & Gospel|Christian Pop",
	1097:     "Christian & Gospel|Christian Rap",
	1098:     "Christian & Gospel|Christian Rock",
	1099:     "Christian & Gospel|Classic Christian",
	1100:     "Christian & Gospel|Contemporary Gospel",
	1101:     "Christian & Gospel|Gospel",
	1103:     "Christian & Gospel|Praise & Worship",
	1104:     "Christian & Gospel|Southern Gospel",
	1105:     "Christian & Gospel|Traditional Gospel",
	1106:     "Jazz|Avant-Garde Jazz",
	1107:     "Jazz|Contemporary Jazz",
	1108:     "Jazz|Crossover Jazz",
	1109:     "Jazz|Dixieland",
	1110:     "Jazz|Fusion",
	1111:     "Jazz|Latin Jazz",
	1112:     "Jazz|Mainstream Jazz",
	1113:     "Jazz|Ragtime",
	1114:     "Jazz|Smooth Jazz",
	1115:     "Latino|Latin Jazz",
	1116:     "Latino|Contemporary Latin",
	1117:     "Latino|Pop Latino",
	1118:     "Latino|Raices",
	1119:     "Latino|Urbano latino",
	1120:     "Latino|Baladas y Boleros",
	1121:     "Latino|Rock y Alternativo",
	1122:     "Brazilian",
	1123:     "Latino|Musica Mexicana",
	1124:     "Latino|Musica tropical",
	1125:     "New Age|Environmental",
	1126:     "New Age|Healing",
	1127:     "New Age|Meditation",
	1128:     "New Age|Nature",
	1129:     "New Age|Relaxation",
	1130:     "New Age|Travel",
	1131:     "Pop|Adult Contemporary",
	1132:     "Pop|Britpop",
	1133:     "Pop|Pop/Rock",
	1134:     "Pop|Soft Rock",
	1135:     "Pop|Teen Pop",
	1136:     "R&B/Soul|Contemporary R&B",
	1137:     "R&B/Soul|Disco",
	1138:     "R&B/Soul|Doo Wop",
	1139:     "R&B/Soul|Funk",
	1140:     "R&B/Soul|Motown",
	1141:     "R&B/Soul|Neo-Soul",
	1142:     "R&B/Soul|Quiet Storm",
	1143:     "R&B/Soul|Soul",
	1144:     "Rock|Adult Alternative",
	1145:     "Rock|American Trad Rock",
	1146:     "Rock|Arena Rock",
	1147:     "Rock|Blues-Rock",
	1148:     "Rock|British Invasion",
	1149:     "Rock|Death Metal/Black Metal",
	1150:     "Rock|Glam Rock",
	1151:     "Rock|Hair Metal",
	1152:     "Rock|Hard Rock",
	1153:     "Rock|Metal",
	1154:     "Rock|Jam Bands",
	1155:     "Rock|Prog-Rock/Art Rock",
	1156:     "Rock|Psychedelic",
	1157:     "Rock|Rock & Roll",
	1158:     "Rock|Rockabilly",
	1159:     "Rock|Roots Rock",
	1160:     "Rock|Singer/Songwriter",
	1161:     "Rock|Southern Rock",
	1162:     "Rock|Surf",
	1163:     "Rock|Tex-Mex",
	1165:     "Soundtrack|Foreign Cinema",
	1166:     "Soundtrack|Musicals",
	1167:     "Comedy|Novelty",
	1168:     "Soundtrack|Original Score",
	1169:     "Soundtrack|Soundtrack",
	1171:     "Comedy|Standup Comedy",
	1172:     "Soundtrack|TV Soundtrack",
	1173:     "Vocal|Standards",
	1174:     "Vocal|Traditional Pop",
	1175:     "Jazz|Vocal Jazz",
	1176:     "Vocal|Vocal Pop",
	1177:     "African|Afro-Beat",
	1178:     "African|Afro-Pop",
	1179:     "World|Cajun",
	1180:     "World|Celtic",
	1181:     "World|Celtic Folk",
	1182:     "World|Contemporary Celtic",
	1183:     "Reggae|Modern Dancehall",
	1184:     "World|Drinking Songs",
	1185:     "Indian|Indian Pop",
	1186:     "World|Japanese Pop",
	1187:     "World|Klezmer",
	1188:     "World|Polka",
	1189:     "World|Traditional Celtic",
	1190:     "World|Worldbeat",
	1191:     "World|Zydeco",
	1192:     "Reggae|Roots Reggae",
	1193:     "Reggae|Dub",
	1194:     "Reggae|Ska",
	1195:     "World|Caribbean",
	1196:     "World|South America",
	1197:     "Arabic",
	1198:     "World|North America",
	1199:     "World|Hawaii",
	1200:     "World|Australia",
	1201:     "World|Japan",
	1202:     "World|France",
	1203:     "African",
	1204:     "World|Asia",
	1205:     "World|Europe",
	1206:     "World|South Africa",
	1207:     "Jazz|Hard Bop",
	1208:     "Jazz|Trad Jazz",
	1209:     "Jazz|Cool Jazz",
	1210:     "Blues|Acoustic Blues",
	1211:     "Classical|High Classical",
	1220:     "Brazilian|Axe",
	1221:     "Brazilian|Bossa Nova",
	1222:     "Brazilian|Choro",
	1223:     "Brazilian|Forro",
	1224:     "Brazilian|Frevo",
	1225:     "Brazilian|MPB",
	1226:     "Brazilian|Pagode",
	1227:     "Brazilian|Samba",
	1228:     "Brazilian|Sertanejo",
	1229:     "Brazilian|Baile Funk",
	1230:     "Alternative|Chinese Alt",
	1231:     "Alternative|Korean Indie",
	1232:     "Chinese",
	1233:     "Chinese|Chinese Classical",
	1234:     "Chinese|Chinese Flute",
	1235:     "Chinese|Chinese Opera",
	1236:     "Chinese|Chinese Orchestral",
	1237:     "Chinese|Chinese Regional Folk",
	1238:     "Chinese|Chinese Strings",
	1239:     "Chinese|Taiwanese Folk",
	1240:     "Chinese|Tibetan Native Music",
	1241:     "Hip-Hop/Rap|Chinese Hip-Hop",
	1242:     "Hip-Hop/Rap|Korean Hip-Hop",
	1243:     "Korean",
	1244:     "Korean|Korean Classical",
	1245:     "Korean|Korean Trad Song",
	1246:     "Korean|Korean Trad Instrumental",
	1247:     "Korean|Korean Trad Theater",
	1248:     "Rock|Chinese Rock",
	1249:     "Rock|Korean Rock",
	1250:     "Pop|C-Pop",
	1251:     "Pop|Cantopop/HK-Pop",
	1252:     "Pop|Korean Folk-Pop",
	1253:     "Pop|Mandopop",
	1254:     "Pop|Tai-Pop",
	1255:     "Pop|Malaysian Pop",
	1256:     "Pop|Pinoy Pop",
	1257:     "Pop|Original Pilipino Music",
	1258:     "Pop|Manilla Sound",
	1259:     "Pop|Indo Pop",
	1260:     "Pop|Thai Pop",
	1261:     "Vocal|Trot",
	1262:     "Indian",
	1263:     "Indian|Bollywood",
	1264:     "Indian|Regional Indian|Tamil",
	1265:     "Indian|Regional Indian|Telugu",
	1266:     "Indian|Regional Indian",
	1267:     "Indian|Devotional & Spiritual",
	1268:     "Indian|Sufi",
	1269:     "Indian|Indian Classical",
	1270:     "Russian|Russian Chanson",
	1271:     "World|Dini",
	1272:     "Turkish|Halk",
	1273:     "Turkish|Sanat",
	1274:     "World|Dangdut",
	1275:     "World|Indonesian Religious",
	1276:     "World|Calypso",
	1277:     "World|Soca",
	1278:     "Indian|Ghazals",
	1279:     "Indian|Indian Folk",
	1280:     "Turkish|Arabesque",
	1281:     "African|Afrikaans",
	1282:     "World|Farsi",
	1283:     "World|Israeli",
	1284:     "Arabic|Khaleeji",
	1285:     "Arabic|North African",
	1286:     "Arabic|Arabic Pop",
	1287:     "Arabic|Islamic",
	1288:     "Soundtrack|Sound Effects",
	1289:     "Folk",
	1290:     "Orchestral",
	1291:     "Marching",
	1293:     "Pop|Oldies",
	1294:     "Country|Thai Country",
	1295:     "World|Flamenco",
	1296:     "World|Tango",
	1297:     "World|Fado",
	1298:     "World|Iberia",
	1299:     "Russian",
	1300:     "Turkish",
	100000:   "Christian & Gospel",
	100001:   "Classical|Art Song",
	100002:   "Classical|Brass & Woodwinds",
	100003:   "Classical|Solo Instrumental",
	100004:   "Classical|Contemporary Era",
	100005:   "Classical|Oratorio",
	100006:   "Classical|Cantata",
	100007:   "Classical|Electronic",
	100008:   "Classical|Sacred",
	100009:   "Classical|Guitar",
	100010:   "Classical|Piano",
	100011:   "Classical|Violin",
	100012:   "Classical|Cello",
	100013:   "Classical|Percussion",
	100014:   "Electronic|Dubstep",
	100015:   "Electronic|Bass",
	100016:   "Hip-Hop/Rap|UK Hip-Hop",
	100017:   "Reggae|Lovers Rock",
	100018:   "Alternative|EMO",
	100019:   "Alternative|Pop Punk",
	100020:   "Alternative|Indie Pop",
	100021:   "New Age|Yoga",
	100022:   "Pop|Tribute",
	100023:   "Pop|Shows",
	100024:   "Cuban",
	100025:   "Cuban|Mambo",
	100026:   "Cuban|Chachacha",
	100027:   "Cuban|Guajira",
	100028:   "Cuban|Son",
	100029:   "Cuban|Bolero",
	100030:   "Cuban|Guaracha",
	100031:   "Cuban|Timba",
	100032:   "Soundtrack|Video Game",
	100033:   "Indian|Regional Indian|Punjabi|Punjabi Pop",
	100034:   "Indian|Regional Indian|Bengali|Rabindra Sangeet",
	100035:   "Indian|Regional Indian|Malayalam",
	100036:   "Indian|Regional Indian|Kannada",
	100037:   "Indian|Regional Indian|Marathi",
	100038:   "Indian|Regional Indian|Gujarati",
	100039:   "Indian|Regional Indian|Assamese",
	100040:   "Indian|Regional Indian|Bhojpuri",
	100041:   "Indian|Regional Indian|Haryanvi",
	100042:   "Indian|Regional Indian|Odia",
	100043:   "Indian|Regional Indian|Rajasthani",
	100044:   "Indian|Regional Indian|Urdu",
	100045:   "Indian|Regional Indian|Punjabi",
	100046:   "Indian|Regional Indian|Bengali",
	100047:   "Indian|Indian Classical|Carnatic Classical",
	100048:   "Indian|Indian Classical|Hindustani Classical",
	100049:   "African|Afro House",
	100050:   "African|Afro Soul",
	100051:   "African|Afrobeats",
	100052:   "African|Benga",
	100053:   "African|Bongo-Flava",
	100054:   "African|Coupe-Decale",
	100055:   "African|Gqom",
	100056:   "African|Highlife",
	100057:   "African|Kuduro",
	100058:   "African|Kizomba",
	100059:   "African|Kwaito",
	100060:   "African|Mbalax",
	100061:   "African|Ndombolo",
	100062:   "African|Shangaan Electro",
	100063:   "African|Soukous",
	100064:   "African|Taarab",
	100065:   "African|Zouglou",
	100066:   "Turkish|Ozgun",
	100067:   "Turkish|Fantezi",
	100068:   "Turkish|Religious",
	100069:   "Pop|Turkish Pop",
	100070:   "Rock|Turkish Rock",
	100071:   "Alternative|Turkish Alternative",
	100072:   "Hip-Hop/Rap|Turkish Hip-Hop/Rap",
	100073:   "African|Maskandi",
	100074:   "Russian|Russian Romance",
	100075:   "Russian|Russian Bard",
	100076:   "Russian|Russian Pop",
	100077:   "Russian|Russian Rock",
	100078:   "Russian|Russian Hip-Hop",
	100079:   "Arabic|Levant",
	100080:   "Arabic|Levant|Dabke",
	100081:   "Arabic|Maghreb Rai",
	100082:   "Arabic|Khaleeji|Khaleeji Jalsat",
	100083:   "Arabic|Khaleeji|Khaleeji Shailat",
	100084:   "Tarab",
	100085:   "Tarab|Iraqi Tarab",
	100086:   "Tarab|Egyptian Tarab",
	100087:   "Tarab|Khaleeji Tarab",
	100088:   "Pop|Levant Pop",
	100089:   "Pop|Iraqi Pop",
	100090:   "Pop|Egyptian Pop",
	100091:   "Pop|Maghreb Pop",
	100092:   "Pop|Khaleeji Pop",
	100093:   "Hip-Hop/Rap|Levant Hip-Hop",
	100094:   "Hip-Hop/Rap|Egyptian Hip-Hop",
	100095:   "Hip-Hop/Rap|Maghreb Hip-Hop",
	100096:   "Hip-Hop/Rap|Khaleeji Hip-Hop",
	100097:   "Alternative|Indie Levant",
	100098:   "Alternative|Indie Egyptian",
	100099:   "Alternative|Indie Maghreb",
	100100:   "Electronic|Levant Electronic",
	100101:   "Electronic|Electro-Cha'abi",
	100102:   "Electronic|Maghreb Electronic",
	100103:   "Folk|Iraqi Folk",
	100104:   "Folk|Khaleeji Folk",
	100105:   "Dance|Maghreb Dance",
	50000061: "Spoken Word",
	50000063: "Disney",
	50000064: "French Pop",
	50000066: "German Pop",
	50000068: "German Folk",
}

// Detect PNG image if "implicit" class is used
var pngHeader = []byte{137, 80, 78, 71, 13, 10, 26, 10}

type atomNames map[string]string

func (f atomNames) Name(n string) []string {
	res := make([]string, 1)
	for k, v := range f {
		if v == n {
			res = append(res, k)
		}
	}
	return res
}

// metadataMP4 is the implementation of Metadata for MP4 tag (atom) data.
type metadataMP4 struct {
	fileType FileType
	data     map[string]interface{}
}

// ReadAtoms reads MP4 metadata atoms from the io.ReadSeeker into a Metadata, returning
// non-nil error if there was a problem.
func ReadAtoms(r io.ReadSeeker) (Metadata, error) {
	m := metadataMP4{
		data:     make(map[string]interface{}),
		fileType: UnknownFileType,
	}
	err := m.readAtoms(r)
	return m, err
}

func (m metadataMP4) readAtoms(r io.ReadSeeker) error {
	for {
		name, size, err := readAtomHeader(r)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch name {
		case "meta":
			// next_item_id (int32)
			_, err := readBytes(r, 4)
			if err != nil {
				return err
			}
			fallthrough

		case "moov", "udta", "ilst":
			return m.readAtoms(r)
		}

		_, ok := atoms[name]
		var data []string
		if name == "----" {
			name, data, err = readCustomAtom(r, size)
			if err != nil {
				return err
			}

			if name != "----" {
				ok = true
				size = 0 // already read data
			}
		}

		if !ok {
			_, err := r.Seek(int64(size-8), io.SeekCurrent)
			if err != nil {
				return err
			}
			continue
		}

		err = m.readAtomData(r, name, size-8, data)
		if err != nil {
			return err
		}
	}
}

func (m metadataMP4) readAtomData(r io.ReadSeeker, name string, size uint32, processedData []string) error {
	var b []byte
	var err error
	var contentType string
	if len(processedData) > 0 {
		b = []byte(strings.Join(processedData, ";")) // add delimiter if multiple data fields
		contentType = "text"
	} else {
		// read the data
		b, err = readBytes(r, uint(size))
		if err != nil {
			return err
		}
		if len(b) < 8 {
			return fmt.Errorf("invalid encoding: expected at least %d bytes, got %d", 8, len(b))
		}

		// "data" + size (4 bytes each)
		b = b[8:]
		if len(b) < 4 {
			return fmt.Errorf("invalid encoding: expected at least %d bytes, for class, got %d", 4, len(b))
		}

		if name == "gnre" {
			m.data[name] = getInt(b[len(b)-1:])
			return nil
		}

		class := getInt(b[1:4])
		var ok bool
		contentType, ok = atomTypes[class]
		if !ok {
			return fmt.Errorf("invalid content type: %v (%x) (%x)", class, b[1:4], b)
		}

		// 4: atom version (1 byte) + atom flags (3 bytes)
		// 4: NULL (usually locale indicator)
		if len(b) < 8 {
			return fmt.Errorf("invalid encoding: expected at least %d bytes, for atom version and flags, got %d", 8, len(b))
		}
		b = b[8:]
	}

	if name == "trkn" || name == "disk" {
		if len(b) < 6 {
			return fmt.Errorf("invalid encoding: expected at least %d bytes, for track and disk numbers, got %d", 6, len(b))
		}

		m.data[name] = int(b[3])
		m.data[name+"_count"] = int(b[5])
		return nil
	}

	if contentType == "implicit" {
		if name == "covr" {
			if bytes.HasPrefix(b, pngHeader) {
				contentType = "png"
			}
			// TODO(dhowden): Detect JPEG formats too (harder).
		}
	}

	var data interface{}
	switch contentType {
	case "implicit":
		if _, ok := atoms[name]; ok {
			return fmt.Errorf("unhandled implicit content type for required atom: %q", name)
		}
		return nil

	case "text":
		data = string(b)

	case "uint8":
		if len(b) < 1 {
			return fmt.Errorf("invalid encoding: expected at least %d bytes, for integer tag data, got %d", 1, len(b))
		}
		data = getInt(b[len(b)-1:])

	case "jpeg", "png":
		data = &Picture{
			Ext:      contentType,
			MIMEType: "image/" + contentType,
			Data:     b,
		}
	}
	m.data[name] = data

	return nil
}

func readAtomHeader(r io.ReadSeeker) (name string, size uint32, err error) {
	err = binary.Read(r, binary.BigEndian, &size)
	if err != nil {
		return
	}
	name, err = readString(r, 4)
	return
}

// Generic atom.
// Should have 3 sub atoms : mean, name and data.
// We check that mean is "com.apple.iTunes" and we use the subname as
// the name, and move to the data atom.
// Data atom could have multiple data values, each with a header.
// If anything goes wrong, we jump at the end of the "----" atom.
func readCustomAtom(r io.ReadSeeker, size uint32) (_ string, data []string, _ error) {
	subNames := make(map[string]string)

	for size > 8 {
		subName, subSize, err := readAtomHeader(r)
		if err != nil {
			return "", nil, err
		}

		// Remove the size of the atom from the size counter
		if size >= subSize {
			size -= subSize
		} else {
			return "", nil, errors.New("--- invalid size")
		}

		b, err := readBytes(r, uint(subSize-8))
		if err != nil {
			return "", nil, err
		}

		if len(b) < 4 {
			return "", nil, fmt.Errorf("invalid encoding: expected at least %d bytes, got %d", 4, len(b))
		}
		switch subName {
		case "mean", "name":
			subNames[subName] = string(b[4:])
		case "data":
			data = append(data, string(b[4:]))
		}
	}

	// there should remain only the header size
	if size != 8 {
		err := errors.New("---- atom out of bounds")
		return "", nil, err
	}

	if subNames["mean"] != "com.apple.iTunes" || subNames["name"] == "" || len(data) == 0 {
		return "----", nil, nil
	}
	return subNames["name"], data, nil
}

func (metadataMP4) Format() Format       { return MP4 }
func (m metadataMP4) FileType() FileType { return m.fileType }

func (m metadataMP4) Raw() map[string]interface{} { return m.data }

func (m metadataMP4) getString(n []string) string {
	for _, k := range n {
		if x, ok := m.data[k]; ok {
			return x.(string)
		}
	}
	return ""
}

func (m metadataMP4) getInt(n []string) int {
	for _, k := range n {
		if x, ok := m.data[k]; ok {
			return x.(int)
		}
	}
	return 0
}

func (m metadataMP4) Title() string {
	return m.getString(atoms.Name("title"))
}

func (m metadataMP4) Artist() string {
	return m.getString(atoms.Name("artist"))
}

func (m metadataMP4) Album() string {
	return m.getString(atoms.Name("album"))
}

func (m metadataMP4) AlbumArtist() string {
	return m.getString(atoms.Name("album_artist"))
}

func (m metadataMP4) Composer() string {
	return m.getString(atoms.Name("composer"))
}

func (m metadataMP4) Genre() string {
	genre := m.getString(atoms.Name("genre"))
	if len(genre) < 1 {
		genreID := m.getInt(atoms.Name("genre ID"))
		if genreID == 0 {
			genreID := m.getInt(atoms.Name("genre ID3v1 ID")) - 1
			genre = id3v1Genres[genreID]
		} else {
			genre = genreIDValues[genreID]
		}
	}
	return genre
}

func (m metadataMP4) Year() int {
	date := m.getString(atoms.Name("year"))
	if len(date) >= 4 {
		year, _ := strconv.Atoi(date[:4])
		return year
	}
	return 0
}

func (m metadataMP4) Track() (int, int) {
	x := m.getInt([]string{"trkn"})
	if n, ok := m.data["trkn_count"]; ok {
		return x, n.(int)
	}
	return x, 0
}

func (m metadataMP4) Disc() (int, int) {
	x := m.getInt([]string{"disk"})
	if n, ok := m.data["disk_count"]; ok {
		return x, n.(int)
	}
	return x, 0
}

func (m metadataMP4) Lyrics() string {
	t, ok := m.data["\xa9lyr"]
	if !ok {
		return ""
	}
	return t.(string)
}

func (m metadataMP4) Comment() string {
	t, ok := m.data["\xa9cmt"]
	if !ok {
		return ""
	}
	return t.(string)
}

func (m metadataMP4) Picture() *Picture {
	v, ok := m.data["covr"]
	if !ok {
		return nil
	}
	p, _ := v.(*Picture)
	return p
}

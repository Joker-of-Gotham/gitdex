# Story 1.2: Run Terminal-First Setup and Environment Diagnostics

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

浣滀负鏂扮殑 Gitdex 鐢ㄦ埛锛?鎴戝笇鏈涢€氳繃缁堢鍐呯殑 setup 娴佺▼瀹屾垚韬唤銆佹潈闄愩€侀粯璁ゅ亸濂藉拰鐜璇婃柇锛?浠ヤ究涓嶇敤绂诲紑缁堢锛屼篃涓嶇敤鐚滄祴鍝噷閰嶇疆閿欒锛屽氨鑳借繘鍏ュ彲宸ヤ綔鐨勫垵濮嬬姸鎬併€?
## Acceptance Criteria

1. **Given** 鐢ㄦ埛绗竴娆″惎鍔?Gitdex  
   **When** 杩愯 setup 娴佺▼  
   **Then** Gitdex 鍦ㄧ粓绔腑寮曞瀹屾垚韬唤閰嶇疆銆侀粯璁よ繍琛屽亸濂斤紝浠ュ強鍏ㄥ眬 / 浠撳簱绾ч厤缃枃浠跺垱寤?2. **Given** 鐢ㄦ埛瀹屾垚鎴栧皾璇?setup  
   **When** 杩愯璇婃柇娴佺▼  
   **Then** Gitdex 浼氭牎楠岃繛鎺ユ€с€佹巿鏉冨噯澶囨儏鍐靛拰蹇呴渶鏈湴宸ュ叿锛屽苟鍦ㄤ换浣曞け璐ユ椂缁欏嚭鍙墽琛屼慨澶嶅缓璁?3. **Given** setup 宸茬敓鎴愰厤缃? 
   **When** 鐢ㄦ埛鍦ㄥ悗缁細璇濅腑浠庣粓绔煡鐪嬮厤缃? 
   **Then** Gitdex 鑳介噸鏂板姞杞藉苟娓呮櫚灞曠ず褰撳墠鐢熸晥閰嶇疆鍙婂叾鏉ユ簮

## Tasks / Subtasks

- [x] 鎵╁睍閰嶇疆鍒嗗眰涓庢寔涔呭寲妯″瀷锛屾敮鎾?global / repo / session / env (AC: 1, 3)
  - [x] 鍦?`internal/platform/config` 涓妸褰撳墠 starter 閰嶇疆鎵╁睍涓哄灞傛ā鍨嬶紝淇濈暀鏃㈡湁閿?`output`銆乣log_level`銆乣profile`銆乣daemon.health_address` 鐨勫吋瀹规€э紝涓嶅仛鐮村潖鎬ф敼鍚?  - [x] 澧炲姞鍏ㄥ眬閰嶇疆榛樿璺緞瑙ｆ瀽涓庝粨搴撶骇閰嶇疆榛樿璺緞瑙ｆ瀽锛屽苟杩斿洖鐢熸晥鏂囦欢璺緞 / 鏉ユ簮鍏冩暟鎹紝渚夸簬鍚庣画 `config show` 涓?`doctor` 鐩存帴澶嶇敤
  - [x] 淇濇寔褰撳墠鈥滃彧鏈夋樉寮忎紶鍏?flag 鎵嶈鐩栭厤缃眰鈥濈殑淇琛屼负锛涗笉瑕佹妸 Cobra 榛樿鍊奸噸鏂板綋浣滅敤鎴疯緭鍏?- [x] 瀹炵幇缁堢浼樺厛鐨?setup 鍏ュ彛涓庨厤缃啓鍏?(AC: 1, 3)
  - [x] 鍦?`internal/cli/command` 涓嬫柊澧?`gitdex init` 鍏ュ彛锛屾敮鎸佺函鏂囨湰浜や簰涓庡彲娴嬭瘯鐨勯潪浜や簰杈撳叆璺緞
  - [x] setup 鑷冲皯瑕嗙洊锛氳韩浠芥ā寮忋€佽緭鍑烘牸寮忓亸濂姐€乸rofile銆佹棩蹇楃骇鍒€佹槸鍚﹀啓鍏ュ叏灞€閰嶇疆銆佹槸鍚﹀啓鍏ヤ粨搴撶骇閰嶇疆锛屼互鍙婂繀瑕佺殑 GitHub App 閰嶇疆鍗犱綅淇℃伅
  - [x] setup 瀹屾垚鍚庣珛鍗抽€氳繃鐪熷疄鍔犺浇璺緞閲嶆柊璇诲彇閰嶇疆锛屽苟杈撳嚭鎽樿銆佹枃浠朵綅缃笌涓嬩竴姝ュ懡浠ゅ缓璁?- [x] 瀹炵幇鍙鐢ㄧ殑鐜璇婃柇涓庨厤缃煡鐪嬭兘鍔?(AC: 2, 3)
  - [x] 鏂板 `gitdex doctor`锛屾鏌ラ厤缃彲璇绘€с€乺epo 涓婁笅鏂囥€丟itHub 杩為€氭€с€佽韩浠介厤缃畬鏁存€т笌蹇呴渶鏈湴宸ュ叿
  - [x] 鏂板 `gitdex config show` 鎴栫瓑浠峰彧璇诲懡浠わ紝灞曠ず褰撳墠鐢熸晥閰嶇疆銆侀厤缃潵婧愪笌鏄惁瀛樺湪 repo-local override
  - [x] 灏嗚瘖鏂€昏緫鎶藉埌鍏变韩鏈嶅姟灞傦紝涓嶈鎶婃鏌ラ€昏緫鐩存帴鍐欐鍦?Cobra handler 涓?- [x] 寤虹珛绋冲畾鐨勮瘖鏂粨鏋滄ā鍨嬶紝渚涘悗缁?TUI / API 澶嶇敤 (AC: 2)
  - [x] 姣忎釜 check 鑷冲皯鏆撮湶绋冲畾瀛楁锛歚id`銆乣status`銆乣summary`銆乣detail`銆乣fix`銆乣source`
  - [x] 鍖哄垎鈥滃皻鏈厤缃€濃€滈厤缃笉瀹屾暣鈥濃€滃凡閰嶇疆浣嗛獙璇佸け璐モ€濃€滈獙璇侀€氳繃鈥濊繖鍑犵被鐘舵€侊紝閬垮厤鎶婃湭閰嶇疆璇姤涓烘垚鍔?  - [x] 鎺堟潈妫€鏌ュ湪鏈?story 涓彧鍋氬埌 GitHub App 鍑嗗搴︿笌杩為€氭€ц瘖鏂紝涓嶄吉閫犵湡瀹?token 鐢宠鎴栧畨瑁呮垚鍔?- [x] 琛ラ綈娴嬭瘯涓庡洖褰掗獙璇侊紝纭繚 setup/doctor 鍙噸澶嶃€佸彲鑷姩鍖?(AC: 1, 2, 3)
  - [x] 涓洪厤缃矾寰勮В鏋愩€佸眰绾т紭鍏堢骇銆侀厤缃啓鍏ヤ笌閰嶇疆鍥炶琛ュ崟鍏冩祴璇?  - [x] 涓?`gitdex init`銆乣gitdex doctor`銆乣gitdex config show` 琛ラ泦鎴愭祴璇曪紝瑕嗙洊 repo 鏍圭洰褰曘€佸祵濂楃洰褰曞拰鏃?repo 涓婁笅鏂囦笁绉嶈繍琛屾柟寮?  - [x] 涓?text 杈撳嚭涓庣粨鏋勫寲杈撳嚭鑷冲皯琛ヤ竴缁勭ǔ瀹氭柇瑷€锛岄伩鍏嶅悗缁?story 鍦ㄨ瘖鏂懡浠や笂鎶婃満鍣ㄥ彲璇昏涔夋墦鏁?- [x] 鎺у埗鏈?story 鑼冨洿锛岄伩鍏嶆彁鍓嶅疄鐜板悗缁兘鍔?(AC: 1, 2, 3)
  - [x] 涓嶅疄鐜扮湡瀹?GitHub App 瀹夎銆丣WT 绛惧彂銆乮nstallation token 鑾峰彇銆亀ebhook 鎺ュ叆鎴?GitHub 鍐欐搷浣?  - [x] 涓嶅疄鐜?repo 鎵弿銆佺姸鎬佹憳瑕佹帹鑽愪换鍔°€乻tructured plan compiler銆乤udit ledger 鎴?rich TUI
  - [x] 涓嶆妸 config/setup 閫昏緫濉炶繘 `cmd/` 鎴?`_bmad-output/`锛涗骇鍝佸疄鐜颁粛鐒跺彧钀藉湪婧愮爜鏍?
## Dev Notes

- Story 1.1 宸茬粡鎻愪緵浜嗗彲宸ヤ綔鐨?Cobra + Viper starter 鍩虹嚎锛屼笖淇繃涓€涓叧閿棶棰橈細鍙湁鏄惧紡浼犲叆鐨?`--output`銆乣--log-level`銆乣--profile` 鎵嶈兘瑕嗙洊閰嶇疆灞傘€係tory 1.2 蹇呴』淇濈暀杩欎釜琛屼负銆?- 褰撳墠 `internal/app/bootstrap` 涓?`internal/platform/config` 浠嶇劧鍋?starter 绾у疄鐜帮細瀹冧滑鍙敮鎸佸崟閰嶇疆鏂囦欢鍏ュ彛锛屼笉瓒充互鐩存帴婊¤冻 PRD 瑕佹眰鐨?`global + repo + session + env` 鍥涘眰妯″瀷銆?- 褰撳墠 `config.ResolveRepoRoot` 鏄€氳繃鍚戜笂鏌ユ壘 `go.mod` 鏉ュ畾浣?Gitdex 宸ヤ綔鍖烘牴鐩綍锛屽畠涓嶆槸闈㈠悜鐢ㄦ埛浠撳簱鐨勬渶缁?repo-context 瑙ｆ瀽鍣ㄣ€係tory 1.2 涓嶈兘鎶婂畠璇綋鎴愮湡瀹炩€滅洰鏍囦粨搴撳彂鐜扳€濊兘鍔涖€?- `gitdex init`銆乣gitdex doctor` 涓?`gitdex config show` 搴旀敮鎸佲€滄棤 repo 涓婁笅鏂団€濈殑鍏ㄥ眬妯″紡锛涢娆?setup 涓嶈兘瑕佹眰鐢ㄦ埛蹇呴』绔欏湪鏌愪釜 Git 浠撳簱閲屻€?- setup 鍚庝笉鑳借惤鍒扮┖缁撴灉銆傝嚦灏戣杩斿洖锛氶厤缃啓鍏ョ粨鏋溿€佽瘖鏂憳瑕併€佷笅涓€姝ュ懡浠ゆ彁绀恒€?
### Technical Requirements

- 缁х画浣跨敤鐜版湁 Go / Cobra / Viper 鍩虹嚎锛屼笉鏇挎崲 CLI 妗嗘灦銆?- 閰嶇疆浼樺厛绾у繀椤讳笌 PRD 鍜?Viper 璇箟瀵归綈锛氭樉寮?session/flag override > environment variables > repo config > global config > built-in defaults銆?- 鍥犱负 Viper 鍗曚釜瀹炰緥鍙敮鎸佽鍙栦竴涓厤缃枃浠讹紝鏈?story 鐨勫灞傞厤缃繀椤绘樉寮忓仛 merge锛涗笉鑳借鐢?`AddConfigPath` 鏈熷緟鑷姩鍙犲姞澶氫釜鏂囦欢銆?- 鍏ㄥ眬閰嶇疆璺緞搴斿熀浜?`os.UserConfigDir()` 鍋氳法骞冲彴瑙ｆ瀽锛岄伩鍏嶇‖缂栫爜鐢ㄦ埛鐩綍銆?- 浠撳簱绾ч厤缃繀椤讳笌鈥滅洰鏍囦粨搴撲笂涓嬫枃鈥濈粦瀹氾紝鑰屼笉鏄粯璁ゅ啓鍥?Gitdex 婧愮爜浠撳簱锛涜嫢娌℃湁 repo 涓婁笅鏂囷紝搴斿厑璁歌烦杩?repo-local config銆?- setup 涓嫢闇€瑕佽褰?GitHub App 韬唤淇℃伅锛屼紭鍏堣褰曟爣璇嗙銆佷富鏈哄湴鍧€銆乮nstallation/app ID銆乸rivate key 璺緞绛夊彲瀹¤瀛楁锛屼笉瑕佽姹傛妸绉侀挜鍘熸枃鐩存帴鍐欒繘閰嶇疆鏂囦欢銆?- 璇婃柇杈撳嚭蹇呴』缁欏嚭鏄庣‘淇寤鸿锛屼笉鑳藉彧鎵撳嵃鈥滃け璐モ€濄€?
### Architecture Compliance

- Cobra command wiring 鐣欏湪 `internal/cli/command/`锛涢厤缃悎骞躲€佽矾寰勮В鏋愩€佽鍐欎笌 source tracking 鐣欏湪 `internal/platform/config/`銆?- 鍏变韩鐨?setup / doctor 涓氬姟閫昏緫搴旀斁鍦?`internal/app/` 鎴栧悓绾у疄鐜板寘涓紝涓嶈鍫嗚繘鍛戒护澶勭悊鍑芥暟銆?- 杩欎竴杞粛鐒舵槸 terminal-first / text-first锛涗笉瑕佸洜涓哄悗缁細鏈?Bubble Tea TUI 灏辨妸褰撳墠 story 缁戝畾鍒?rich TUI銆?- `gitdexd` 浠嶄繚鎸?Story 1.1 鐨?daemon starter 鑱岃矗锛汼tory 1.2 鐨勭敤鎴峰叆鍙ｉ泦涓湪 `gitdex`銆?- 浠讳綍鏂板缁撴瀯鍖栬緭鍑洪兘搴斾负鍚庣画 CLI / TUI / API 鍏辩敤璇箟鍑嗗锛岃€屼笉鏄仛涓€娆℃€ф祴璇曚笓鐢ㄦ牸寮忋€?
### Library / Framework Requirements

- 缁х画浣跨敤 `github.com/spf13/cobra v1.10.2` 鎵╁睍鍛戒护鏍戙€?- 缁х画浣跨敤浠撳簱鐜版湁鐨?`github.com/spf13/viper v1.21.0`锛涘灞傞厤缃涔堥€氳繃澶氬疄渚?merge锛岃涔堥€氳繃鍙楁帶璇诲彇椤哄簭鍙犲姞锛岄伩鍏嶅紩鍏ョ浜屽閰嶇疆妗嗘灦銆?- 濡傛灉瑕佹妸 flag 鏄犲皠鍥為厤缃鍙栵紝浼樺厛娌跨敤宸叉湁鈥滄樉寮?flag 鎵嶈鐩栤€濈殑妯″紡锛屾垨杩佺Щ鍒?`BindPFlag` + `Changed` 璇箟娓呮櫚鐨勫疄鐜帮紝浣嗗繀椤绘湁鍥炲綊娴嬭瘯瀹堜綇 precedence銆?- 璇婃柇涓闇€鍋氱綉缁滄帰娲伙紝搴旀敞鍏ュ彲鏇挎崲 client / checker锛岄伩鍏嶆祴璇曚緷璧栫湡瀹炲叕缃戙€?
### File Structure Requirements

- 鏈?story 棰勮涓昏瑙﹁揪涓嬪垪鍖哄煙锛?  - `internal/cli/command/`
  - `internal/app/bootstrap/`
  - `internal/platform/config/`
  - `internal/cli/output/`
  - `configs/gitdex.example.yaml`
  - `test/integration/`
  - `test/conformance/`
- 濡傞渶鏂板鍏变韩鏈嶅姟锛屼紭鍏堣€冭檻锛?  - `internal/app/setup/`
  - `internal/app/doctor/`
- 涓嶈鍦?`cmd/gitdex/main.go` 涓啓涓氬姟閫昏緫锛沵ain 浠嶅彧璐熻矗璋冪敤 command tree銆?
### Testing Requirements

- 鍗曞厓娴嬭瘯蹇呴』瑕嗙洊锛?  - 澶氬眰閰嶇疆浼樺厛绾?  - 鍏ㄥ眬 / repo 閰嶇疆璺緞瑙ｆ瀽
  - 閰嶇疆鍐欏叆鍚庣殑鍐嶅姞杞?  - 璇婃柇缁撴灉鐘舵€佹槧灏?- 闆嗘垚娴嬭瘯蹇呴』瑕嗙洊锛?  - 棣栨 setup 鎴愬姛鍐欏叆鍏ㄥ眬閰嶇疆
  - 鍦ㄧ湡瀹?repo 瀛愮洰褰曟墽琛?setup / doctor / config show
  - 鏃?repo 涓婁笅鏂囨椂璧?global-only 妯″紡
  - 韬唤鏈厤缃笌閰嶇疆涓嶅畬鏁存椂锛宒octor 缁欏嚭鍙墽琛屼慨澶嶅缓璁?- 娴嬭瘯涓嶅緱鍐欏叆鐪熷疄鐢ㄦ埛鐩綍锛涙墍鏈夊叏灞€閰嶇疆璺緞閮借閲嶅畾鍚戝埌涓存椂鐩綍銆?- 杩炴帴鎬ф祴璇曚笉寰楃洿鎺ヤ緷璧栫湡瀹?`api.github.com`锛涘簲閫氳繃鍙敞鍏ユ帰娲诲櫒鎴栧彲鎺?endpoint fixture 淇濇寔绋冲畾銆?- 鏈?story 瀹屾垚鏃惰嚦灏戣窇閫氾細
  - `go test ./...`
  - `go run ./cmd/gitdex --help`
  - `go run ./cmd/gitdex init --help`
  - `go run ./cmd/gitdex doctor`
  - `go run ./cmd/gitdex config show`
  - `go run ./cmd/gitdexd run`
  - `golangci-lint run`

### Previous Story Intelligence

- Story 1.1 宸茶惤鍦扮殑 starter 缁撴瀯鍖呮嫭锛氬弻鍏ュ彛浜岃繘鍒躲€丆obra 鍛戒护鏍戙€乂iper 閰嶇疆鍔犺浇銆乧ompletion銆乨aemon stub銆侀厤缃笌 conformance / integration 娴嬭瘯楠ㄦ灦銆?- Story 1.1 鐨?review 淇杩?4 涓鏈?story 鐩存帴鐩稿叧鐨勯棶棰橈細
  - flag 榛樿鍊间笉鑳藉帇鎺?env / config
  - 鐗堟湰鍙疯鍙敞鍏?  - Go baseline 閿佸畾鍒?`1.26.1`
  - 宓屽鐩綍娴嬭瘯蹇呴』鐪熷疄杩涘叆瀛愮洰褰曚笖鍙竻鐞?- 鍚庣画瀹炵幇搴旂洿鎺ュ鐢?Story 1.1 鐨勬祴璇曟€濊矾锛氱敤涓存椂鐩綍銆佹樉寮忚緭鍏ャ€佺湡瀹炲懡浠ゆ墽琛屽拰 source-level 鍥炲綊鏂█瀹堜綇 CLI 琛屼负銆?- 褰撳墠浠撳簱涓嶆槸 Git working tree锛屽洜姝ゆ病鏈?commit history 鍙敤浜庨澶栨ā寮忔彁鐐硷紱鏈?story 涓昏缁ф壙涓婁竴鏁呬簨鏂囦欢涓殑瀹炵幇绾︽潫銆?
### Latest Technical Validation

- 鎴嚦 2026-03-18锛孏o 瀹樻柟 release history 浠嶅垪鍑?`go1.26.1`锛屽苟璇存槑鍏朵簬 2026-03-05 鍙戝竷锛涚户缁攣瀹?`Go 1.26.1` 鍚堢悊銆?- 鎴嚦 2026-03-18锛孋obra 瀹樻柟浠撳簱 / release 椤甸潰浠嶆樉绀?`v1.10.2` 涓烘渶鏂扮増鏈紱瀹樻柟 shell completion 鎸囧崡缁х画瑕嗙洊 `bash`銆乣zsh`銆乣fish`銆乣powershell`銆?- 浠撳簱褰撳墠宸插浐瀹?`github.com/spf13/viper v1.21.0`锛沄iper 瀹樻柟 README 浠嶈鏄?precedence 涓?`Set > flags > env > config file > external store > defaults`锛屼笖鍗曚釜瀹炰緥鍙敮鎸佷竴涓厤缃枃浠讹紝闇€瑕佹樉寮?merge 澶氬眰閰嶇疆銆?- GitHub 瀹樻柟 GitHub App 鏂囨。浠嶈姹傞€氳繃 `JWT -> installation access token` 璺緞瀹屾垚瀹夎绾ц璇侊紝涓?installation access token 榛樿 1 灏忔椂杩囨湡锛涘洜姝ゆ湰 story 鍙仛 GitHub App 瀵煎悜鐨勯厤缃笌璇婃柇鍑嗗锛屼笉鎻愬墠瀹炵幇鐪熷疄鎺堟潈娴併€?
### Project Context Reference

- 褰撳墠宸ヤ綔鍖烘湭妫€娴嬪埌 `project-context.md`銆?- 鏈?story 鐨勬潈濞佷笂涓嬫枃鏉ユ簮涓猴細`epics.md`銆乣prd.md`銆乣architecture.md`銆乣ux-design-specification.md` 涓?Story 1.1 瀹炵幇璁板綍銆?
### Project Structure Notes

- Story 1.2 鏄粠 starter 杩涘叆鈥滅湡瀹?operator onboarding surface鈥濈殑绗竴姝ワ紝浣嗕粛灞炰簬 Phase 1 鐨勭獎鑼冨洿鍩虹璁炬柦锛屼笉搴旇秺绾у疄鐜?repo orchestration銆乸olicy engine 鎴?task state machine銆?- 鐢变簬 UX 鏄庣‘瑕佹眰 `setup 鍚庝笉鏄┖鐣岄潰`锛宻etup 缁撴灉椤?/ 鏂囨湰鎽樿瑕佽褰撴垚浜у搧琛屼负鏉ヨ璁★紝鑰屼笉鏄复鏃跺紑鍙戞棩蹇椼€?- 鐢变簬鏋舵瀯瑕佹眰 Windows / Linux / macOS 缁熶竴鎿嶄綔璇箟锛岃矾寰勩€佹帰娴嬨€佹彁绀烘枃妗堝拰娴嬭瘯澶瑰叿閮借閬垮厤 shell 涓撳睘鍐欐硶銆?
### Non-Goals / Scope Guardrails

- 涓嶅疄鐜扮湡瀹?GitHub App 娉ㄥ唽銆佸畨瑁呫€丣WT 鐢熸垚銆乮nstallation token 鐢宠鎴?GitHub API 鍐欏叆銆?- 涓嶅疄鐜伴娆?repo 鎵弿銆乺epo state summary銆佹帹鑽愪换鍔℃垨 structured plan 鐢熸垚銆?- 涓嶅疄鐜版寔涔呭寲浠诲姟鐘舵€併€佸璁¤处鏈€乤pproval 璺敱銆乸olicy bundle 鐢熸晥鎴?background reconciliation銆?- 涓嶅疄鐜?rich TUI锛涘綋鍓嶅彧瑕佹眰 text-first CLI 鍜屽悗缁彲澶嶇敤鐨勬暟鎹涔夈€?- 涓嶄负浜嗏€滃厛璺戣捣鏉モ€濊€屽紩鍏ラ暱鏈?PAT 浣滀负榛樿韬唤妯″紡銆?
### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-12-Run-Terminal-First-Setup-and-Environment-Diagnostics-FR44-FR45-FR48]
- [Source: _bmad-output/planning-artifacts/prd.md#Command-Structure]
- [Source: _bmad-output/planning-artifacts/prd.md#Config-Schema]
- [Source: _bmad-output/planning-artifacts/prd.md#Configuration-Onboarding--Operator-Enablement]
- [Source: _bmad-output/planning-artifacts/prd.md#Journey-1-鐙珛缁存姢鑰呮妸-Gitdex-鍙樻垚榛樿缁堢鍏ュ彛]
- [Source: _bmad-output/planning-artifacts/architecture.md#Selected-Starter-Cobra-Based-Go-Workspace-Foundation]
- [Source: _bmad-output/planning-artifacts/architecture.md#Project-Structure--Boundaries]
- [Source: _bmad-output/planning-artifacts/architecture.md#Implementation-Handoff]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Journey-1-鏂扮敤鎴蜂粠棣栨-setup-鍒扮涓€娆″€煎緱淇濈暀鐨勬垚鍔熶綋楠宂
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Component-Implementation-Strategy]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Testing-Strategy]
- [Source: _bmad-output/implementation-artifacts/1-1-set-up-initial-project-from-starter-template.md]
- [External: https://go.dev/doc/devel/release]
- [External: https://github.com/spf13/cobra]
- [External: https://github.com/spf13/cobra/releases/tag/v1.10.2]
- [External: https://cobra.dev/docs/how-to-guides/shell-completion/]
- [External: https://github.com/spf13/viper]
- [External: https://github.com/spf13/viper/releases/tag/v1.21.0]
- [External: https://docs.github.com/apps/creating-github-apps/registering-a-github-app/registering-a-github-app]
- [External: https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-an-installation-access-token-for-a-github-app]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- User explicitly requested Story `1.2`.
- Sprint tracking already showed `1-2-run-terminal-first-setup-and-environment-diagnostics` as the next backlog item.
- Previous story file `1-1-set-up-initial-project-from-starter-template.md` was loaded and mined for starter constraints, review fixes, and test patterns.
- No `.git` working tree was detected, so commit-history intelligence was skipped.
- No `project-context.md` was found in the workspace; create-story context came from epics, PRD, architecture, UX, and Story 1.1.
- Latest technical validation was refreshed against official Go, Cobra, Viper, and GitHub Docs sources on 2026-03-18.
- Local validation completed with `go test ./...`, `golangci-lint run`, `go run ./cmd/gitdex --help`, `go run ./cmd/gitdex init --help`, `go run ./cmd/gitdex doctor`, `go run ./cmd/gitdex config show`, and `go run ./cmd/gitdexd run`.
- Self-review loop closed after tightening onboarding docs, example config coverage, text output assertions, and active config file visibility in `config show`.

### Completion Notes List

- Implemented layered global/repo/env/flag config loading with source tracking, repository-context detection, and config file persistence helpers.
- Added terminal-first onboarding commands: `gitdex init`, `gitdex doctor`, and `gitdex config show`, backed by shared `setup` and `doctor` service packages.
- Expanded onboarding tests across unit, integration, and conformance layers, including global-only mode, nested repository execution, example config coverage, and stable text/structured output assertions.
- Updated README and example config so the published starter surface matches Story 1.2 behavior.

### File List

- _bmad-output/implementation-artifacts/1-2-run-terminal-first-setup-and-environment-diagnostics.md
- _bmad-output/implementation-artifacts/sprint-status.yaml
- README.md
- configs/gitdex.example.yaml
- internal/app/bootstrap/bootstrap.go
- internal/app/doctor/doctor.go
- internal/app/doctor/doctor_test.go
- internal/app/setup/setup.go
- internal/app/setup/setup_test.go
- internal/cli/command/config_command.go
- internal/cli/command/doctor.go
- internal/cli/command/helpers.go
- internal/cli/command/init.go
- internal/cli/command/render_test.go
- internal/cli/command/root.go
- internal/cli/command/root_test.go
- internal/cli/output/format.go
- internal/cli/output/format_test.go
- internal/platform/config/config.go
- internal/platform/config/config_test.go
- test/conformance/starter_skeleton_test.go
- test/integration/onboarding_commands_test.go

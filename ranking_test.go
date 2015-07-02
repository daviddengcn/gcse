package gcse

import (
	"strings"
	"testing"

	"github.com/golangplus/testing/assert"
)

func TestEffectiveImported(t *testing.T) {
	pkg := "labix.org/v2/mgo/bson"
	imported := strings.Fields(
		`bitbucket.org/phlyingpenguin/collectinator
bitbucket.org/phlyingpenguin/collectinator/account
bitbucket.org/phlyingpenguin/collectinator/blog
bitbucket.org/phlyingpenguin/collectinator/common
code.google.com/p/sadbox/sessions/mongodb
git.300brand.com/coverage
git.300brand.com/coverage/storage/mongo
github.com/GeertJohan/outyet
github.com/MG-RAST/AWE/core
github.com/MG-RAST/Shock/shock-server
github.com/MG-RAST/Shock/shock-server/controller/node
github.com/MG-RAST/Shock/shock-server/node
github.com/MG-RAST/Shock/shock-server/preauth
github.com/MG-RAST/Shock/shock-server/user
github.com/abiosoft/gopages-sample/store
github.com/adnaan/hamster
github.com/alouca/MongoQueue
github.com/araddon/loges
github.com/araddon/mgou
github.com/arbaal/go-gridfs-serve
github.com/athom/tenpu
github.com/bketelsen/skynet/client
github.com/bketelsen/skynet/rpc/bsonrpc
github.com/bketelsen/skynet/service
github.com/cgrates/cgrates/rater
github.com/chrissexton/alepale/bot
github.com/chrissexton/alepale/plugins
github.com/drevell/mgou
github.com/edsrzf/mgo
github.com/emicklei/landskape/dao
github.com/fluffle/sp0rkle/collections/conf
github.com/fluffle/sp0rkle/collections/factoids
github.com/fluffle/sp0rkle/collections/karma
github.com/fluffle/sp0rkle/collections/markov
github.com/fluffle/sp0rkle/collections/quotes
github.com/fluffle/sp0rkle/collections/reminders
github.com/fluffle/sp0rkle/collections/seen
github.com/fluffle/sp0rkle/collections/stats
github.com/fluffle/sp0rkle/collections/urls
github.com/fluffle/sp0rkle/drivers/factdriver
github.com/fluffle/sp0rkle/drivers/quotedriver
github.com/fluffle/sp0rkle/drivers/reminddriver
github.com/fluffle/sp0rkle/drivers/urldriver
github.com/globocom/gandalf/api
github.com/globocom/gandalf/repository
github.com/globocom/gandalf/user
github.com/globocom/mongoapi
github.com/globocom/tsuru/api
github.com/globocom/tsuru/app
github.com/globocom/tsuru/auth
github.com/globocom/tsuru/collector
github.com/globocom/tsuru/provision/docker
github.com/globocom/tsuru/provision/juju
github.com/globocom/tsuru/provision/lxc
github.com/globocom/tsuru/quota
github.com/globocom/tsuru/service
github.com/godfried/impendulo/db
github.com/godfried/impendulo/processing
github.com/godfried/impendulo/processing/monitor
github.com/godfried/impendulo/project
github.com/godfried/impendulo/server
github.com/godfried/impendulo/server/web
github.com/godfried/impendulo/tool
github.com/godfried/impendulo/tool/javac
github.com/godfried/impendulo/tool/jpf
github.com/godfried/impendulo/tool/junit
github.com/godfried/impendulo/util
github.com/gosexy/db/mongo
github.com/gregworley/koalab-golang-api
github.com/isaiah/tsuru_service
github.com/jasonmoo/gearman-go/client
github.com/jbaikge/coverage
github.com/jbaikge/est
github.com/jmcvetta/jfu
github.com/jmoiron/monet/app
github.com/jmoiron/monet/blog
github.com/jmoiron/monet/db
github.com/jmoiron/monet/gallery
github.com/johnwesonga/gotodolist
github.com/jordanorelli/go-instagram
github.com/jordanorelli/twitter
github.com/kidstuff/mtoy
github.com/kidstuff/mtoy/mgoauth
github.com/kidstuff/mtoy/mgosessions
github.com/liudian/mogogo/src/mogogo
github.com/lukegb/irclogsme
github.com/lukegb/irclogsme/logger
github.com/lukegb/irclogsme/server
github.com/mdennebaum/mgomodel
github.com/melvinmt/startupreader.com
github.com/mikespook/gearman-go/client
github.com/miraclesu/service
github.com/monnand/bully
github.com/mschoch/tuq/datasources/mongodb
github.com/msurdi/alf/db
github.com/netbrain/gonk/examples/authentication/app/role
github.com/netbrain/gonk/examples/authentication/app/user
github.com/nono/koalab-golang-api
github.com/nstott/mongobench
github.com/nvcnvn/glog
github.com/nvcnvn/glog/dbctx
github.com/nvcnvn/gorms
github.com/nvcnvn/gorms/dbctx
github.com/openvn/toys/secure/membership
github.com/openvn/toys/secure/membership/sessions
github.com/opesun/hypecms/model/basic
github.com/opesun/hypecms/model/patterns
github.com/opesun/hypecms/model/scut
github.com/opesun/hypecms/modules/admin/model
github.com/opesun/hypecms/modules/bootstrap
github.com/opesun/hypecms/modules/bootstrap/model
github.com/opesun/hypecms/modules/content
github.com/opesun/hypecms/modules/content/model
github.com/opesun/hypecms/modules/custom_actions
github.com/opesun/hypecms/modules/custom_actions/model
github.com/opesun/hypecms/modules/display_editor
github.com/opesun/hypecms/modules/display_editor/model
github.com/opesun/hypecms/modules/skeleton
github.com/opesun/hypecms/modules/template_editor
github.com/opesun/hypecms/modules/template_editor/model
github.com/opesun/hypecms/modules/user/model
github.com/opesun/nocrud/frame/impl/set/mongodb
github.com/opesun/nocrud/frame/misc/convert
github.com/opesun/nocrud/modules/fulltext
github.com/opesun/resolver
github.com/pavel-paulau/blurr/databases
github.com/pjvds/httpcallback.io/data/mongo
github.com/pjvds/httpcallback.io/model
github.com/prinsmike/GoVHostLog
github.com/prinsmike/gohome
github.com/reiver/turtledq
github.com/retzkek/transfat
github.com/rif/gocmd
github.com/rwynn/gtm
github.com/scottcagno/netkit
github.com/scottferg/goat
github.com/shawnps/mappuri
github.com/skelterjohn/bsonrpc
github.com/stretchr/codecs/bson
github.com/sunfmin/batchbuy/model
github.com/sunfmin/mgodb
github.com/sunfmin/tenpu/gridfs
github.com/sunfmin/tenpu/thumbnails
github.com/surma/importalias
github.com/tanema/mgorx
github.com/trevex/golem_examples
github.com/ungerik/go-start/model
github.com/ungerik/go-start/mongo
github.com/ungerik/go-start/mongomedia
github.com/ungerik/go-start/user
github.com/vbatts/imgsrv
github.com/wendyeq/iweb
github.com/wesnow/qufadai/src
github.com/xing4git/chirp/dao
github.com/xing4git/chirp/dao/redisdao
github.com/xing4git/chirp/model
github.com/xing4git/chirp/service/feedservice
github.com/zeebo/est
github.com/zeebo/goci/app/entities
github.com/zeebo/goci/app/frontend
github.com/zeebo/goci/app/httputil
github.com/zeebo/goci/app/notifications
github.com/zeebo/goci/app/response
github.com/zeebo/goci/app/tracker
github.com/zeebo/goci/app/workqueue
github.com/zeebo/gostbook
labix.org/v2/mgo
labix.org/v2/mgo/txn
launchpad.net/hockeypuck/mgo
launchpad.net/juju-core/charm
launchpad.net/juju-core/state
launchpad.net/juju-core/state/presence
launchpad.net/juju-core/state/watcher
launchpad.net/juju-core/store
launchpad.net/juju-core/version
launchpad.net/mgo/v2`)
	author := AuthorOfPackage(pkg)
	project := ProjectOfPackage(pkg)
	t.Logf("pkg: %s, author: %s, project: %s", pkg, author, project)
	_ = imported
	cnt := effectiveImported(imported, author, project)
	t.Logf("cnt: %f", cnt)

	assert.ValueShould(t, "cnt", cnt, cnt <= 100, "> 100: effectiveImported failed!")
}

func TestEffectiveImported_projsame(t *testing.T) {
	pkg := "github.com/dotcloud/docker/term"
	imported := strings.Fields(
		`github.com/AsherBond/docker
github.com/ChaosCloud/docker
github.com/CodeNow/docker
github.com/DanielBryan/docker
github.com/Jukkrapong/docker
github.com/ToothlessGear/docker
github.com/Vladimiroff/docker
github.com/ZeissS/docker
github.com/amaudy/docker
github.com/anachronistic/docker
github.com/apatil/docker-cpuset-cpus
github.com/apatil/docker-lxc-options
github.com/aybabtme/docker
github.com/bdon/docker
github.com/benoitc/docker
github.com/billyoung/docker
github.com/bits/docker
github.com/bpo/docker
github.com/bradobro/docker
github.com/c4milo/docker
github.com/calavera/docker
github.com/carlosdp/docker
github.com/cespare/docker
github.com/crosbymichael/docker
github.com/dhrp/docker
github.com/dillera/docker
github.com/dlintw/docker
github.com/dotcloud/docker
github.com/dr-strangecode/docker
github.com/dsissitka/docker
github.com/dynport/docker
github.com/ehazlett/docker
github.com/errnoh/docker
github.com/fmd/docker
github.com/fsouza/docker
github.com/fsouza/go-dockerclient
github.com/gaffo/docker
github.com/gale320/docker
github.com/hantuo/docker
github.com/hukeli/docker
github.com/irr/docker
github.com/ismell/docker
github.com/jaepil/docker
github.com/jamtur01/docker
github.com/jbardin/docker
github.com/jmcvetta/docker
github.com/johnbellone/docker
github.com/johnnydtan/docker
github.com/junk16/docker
github.com/justone/docker
github.com/kencochrane/docker
github.com/kisielk/docker
github.com/kmindg/docker
github.com/kpelykh/docker
github.com/kstaken/docker
github.com/lopter/docker
github.com/mars9/docker
github.com/maxhodak/docker
github.com/metalivedev/docker
github.com/mewpkg/docker
github.com/mhennings/docker
github.com/mindreframer/docker
github.com/monnand/docker
github.com/ndarilek/docker
github.com/nickstenning/docker
github.com/offby1/docker
github.com/ooyala/docker
github.com/oss17888/docker
github.com/petar/gocircuit-docker
github.com/philips/docker
github.com/pjvds/docker
github.com/rhoml/docker
github.com/richo/docker
github.com/ryfow/docker
github.com/sabzil/docker
github.com/shin-/docker
github.com/silpion/docker
github.com/sinhalabs/docker
github.com/sleekslush/docker
github.com/sridatta/docker
github.com/stevedomin/docker
github.com/steveruckdashel/docker
github.com/stfp/docker
github.com/synack/docker
github.com/timcubb/docker
github.com/titanous/docker
github.com/twmb/docker
github.com/unclejack/docker
github.com/vagmi/docker
github.com/zimbatm/docker
github.com/zsol/docker`)
	author := AuthorOfPackage(pkg)
	project := ProjectOfPackage(pkg)
	t.Logf("pkg: %s, author: %s, project: %s", pkg, author, project)
	_ = imported
	cnt := effectiveImported(imported, author, project)
	t.Logf("cnt: %f", cnt)

	assert.ValueShould(t, "cnt", cnt, cnt <= 10, "> 10: effectiveImported failed!")
}

func TestProjectOfPackage(t *testing.T) {
	PKG_PRJ := []string{
		`github.com/AsherBond/docker`, `docker`,
		`gopkg.in/redis.v1`, `redis`,
		`gopkg.in/inconshreveable/log15.v2`, `log15`,
		`gopkg.in/fatih/v0/set`, `gopkg.in`,
	}

	for i := 0; i < len(PKG_PRJ); i += 2 {
		assert.Equal(t, "project of "+PKG_PRJ[i], ProjectOfPackage(PKG_PRJ[i]), PKG_PRJ[i+1])
	}
}

func TestAuthorOfPackage(t *testing.T) {
	PKG_AUTHOR := []string{
		`github.com/AsherBond/docker`, `AsherBond`,
		`gopkg.in/redis.v1`, `go-redis`,
		`gopkg.in/inconshreveable/log15.v2`, `inconshreveable`,
		`gopkg.in/fatih/v0/set`, `fatih`,
	}

	for i := 0; i < len(PKG_AUTHOR); i += 2 {
		assert.Equal(t, "author of "+PKG_AUTHOR[i], AuthorOfPackage(PKG_AUTHOR[i]), PKG_AUTHOR[i+1])
	}
}

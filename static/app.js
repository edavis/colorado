"use strict";

function getFavicon(url) {
    var a = document.createElement('a');
    a.href = url;
    return ("http://www.google.com/s2/favicons?domain=" + (a.hostname || 'example.com'));
}

function parseWhen(s) {
    var formats = [
        'ddd, DD MMM YYYY HH:mm:ss ZZ',
        'ddd, DD MMM YYYY HH:mm:ss Z',
    ];
    return moment.utc(s, formats).local();
}

var RiverList = React.createClass({
    fetchRiver: function() {
        // Skip update if user has scrolled down any. This prevents
        // the river from jumping around with new updates as you try
        // to read it.
        if (window.scrollY > 0) {
            return;
        }

        $.ajax({
            url: this.state.url,
            dataType: 'jsonp',
            jsonp: false,
            jsonpCallback: 'onGetRiverStream',
            success: function(data) {
                this.setState({
                    feeds: data.updatedFeeds.updatedFeed,
                    title: data.metadata.title,
                    description: data.metadata.description,
                });
                if (data.metadata.title && data.metadata.description) {
                    document.title = data.metadata.title + ' | ' + data.metadata.description;
                } else if (data.metadata.title) {
                    document.title = data.metadata.title;
                } else {
                    document.title = 'river.js feed';
                }
            }.bind(this),
            error: function(xhr, status, err) {
                console.error(this.state.url, status, err.toString());
            }.bind(this)
        });
    },
    getInitialState: function() {
        return {
            feeds: [],
            title: '',
            url: 'river'
        };
    },
    componentDidMount: function() {
        this.fetchRiver();
        setInterval(this.fetchRiver, 60 * 1000);
    },
    render: function() {
        var that = this;
        var feeds = this.state.feeds.map(function(feed) {
            return <RiverFeed key={feed.whenLastUpdate + feed.feedUrl} feed={feed} />;
        });
        var loading = <div className="loading"><p>Loading&hellip;</p><img src="/static/ajax-loader.gif"></img></div>;
        return (
            <div className="riverContainer">
                <h1 className="title">{this.state.title}</h1>
                {this.state.description ? <h2 className="description">{this.state.description}</h2> : ''}
                <div className="riverList">
                    {this.state.feeds.length ? feeds : loading}
                </div>
            </div>
        );
    }
});

var RiverFeed = React.createClass({
    render: function() {
        var items = this.props.feed.item.map(function(item) {
            return (
                <RiverItem key={item.id} item={item} />
            );
        });
        var whenLastUpdate = parseWhen(this.props.feed.whenLastUpdate).format('h:mm A; DD MMM');
        var favicon = getFavicon(this.props.feed.websiteUrl);
        return (
            <div className="riverFeed">
                <div className="riverHeader">
                    <div className="updateInfo">
                        {whenLastUpdate}
                    </div>
                    <div className="feedInfo">
                        <img className="favicon" src={favicon}></img>
                        <a className="feedTitle" href={this.props.feed.websiteUrl} dangerouslySetInnerHTML={{__html: this.props.feed.feedTitle }} />&nbsp;
                        <a className="feedUrl" href={this.props.feed.feedUrl}>(Feed)</a>
                    </div>
                </div>
                <div className="riverItems">
                    {items}
                </div>
            </div>
        );
    }
});

var RiverItem = React.createClass({
    render: function() {
        var whenAgo = parseWhen(this.props.item.pubDate).fromNow();
        return (
            <div className="riverItem">
                <div className="itemTitle"><a target="_blank" href={this.props.item.link} dangerouslySetInnerHTML={{__html: this.props.item.title || this.props.item.body}} /></div>
                <div className="itemBody" dangerouslySetInnerHTML={{__html: this.props.item.body }} />
                <div className="itemMeta">
                    <span className="whenAgo">{whenAgo}</span>
                    {this.props.item.comments && <span className="commentsUrl">&nbsp;&bull;&nbsp;<a target="_blank" href={this.props.item.comments}>Comments</a></span>}
                </div>
            </div>
        );
    }
});

ReactDOM.render( <RiverList />, document.getElementById('app') );

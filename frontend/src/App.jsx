import {useState} from 'react';
import logo from './assets/images/logo-universal.png';
import './App.css';
import {LinkPlex, GetStatus, GetServers, SetServer, IsAuthorized} from "../wailsjs/go/main/App";

function App() {
    const [statusText, setstatusText] = useState("");
    const [servers, setServers] = useState([]);
    const updateServers = (e) => setServers(e);
    const updateStatusText = (result) => setstatusText(result);

    var authorized;

    status();
    checkAuth();

    function linkPlex() {
        LinkPlex();
    }

    function status() {
        setInterval(()=> {
            GetStatus().then((value) => {updateStatusText(value)});
        },1000);
    }

    function checkAuth() {
        IsAuthorized().then((value) => {
            authorized = value;
            console.log(value);
            if (value) {
                loadServers();
            } else {
                setTimeout(checkAuth(),1000);
            }
        });
    }


    function loadServers() {
        GetServers().then((value) => {
            console.log(value);
            updateServers(value);
            var selectBox = document.getElementById('servers');

            for(var i = 0, l = servers.length; i < l; i++){
                var option = servers[i];
                selectBox.options.add( new Option(option, i, false) );
            }
            selectBox.classList.remove("hidden")
        });
    }

    return (
        <div id="App">
            <div id="result" className="result">{statusText}</div>
            <button className='btn' onClick={linkPlex}>Link Plex</button>
            {/* <div id="input" className="input-box">
                <input id="name" className="input" onChange={updateName} autoComplete="off" name="input" type="text"/>
            </div> */}
            <select className="hidden" id="servers">
            </select>
        </div>
    )
}

export default App

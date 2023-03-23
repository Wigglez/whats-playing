import {useState,useCallback,useEffect} from 'react';
import logo from './assets/images/logo-universal.png';
import './App.css';
import {LinkPlex, GetStatus, GetServers, SetServer, IsAuthorized} from "../wailsjs/go/main/App";

function App() {
    const [statusText, setstatusText] = useState("");
    const [servers, setServers] = useState([]);
    const [authorized, setAuthorized] = useState(null);

    const loadServers = useCallback(() => {
        GetServers().then((value) => {
            console.log(value);
            if (value != null) {
              setServers(value);
            }
        });
    }, [setServers]);

    useEffect(() => {
        let timeout;

        function checkAuth() {
            IsAuthorized().then((value) => {
                setAuthorized(value);
                console.log(value);
                if (value) {
                    loadServers();
                } else {
                    timeout = setTimeout(checkAuth(),1000);
                }
            });
        }

        checkAuth();

        return () => {
            // clean up timeout refs when component unmounts for whatever reason, and don't recreate unless the component remounts
            timeout && clearTimeout(timeout);
        };
    }, [loadServers, setAuthorized]);

    useEffect(() => {
        const interval = setInterval(()=> {
            GetStatus().then((value) => {setstatusText(value)});
        },1000);

        return () => {
            // clean up interval refs
            clearInterval(interval);
        };
    }, [setstatusText]);

    function linkPlex() {
        LinkPlex();
    }

    function setServer(event) {
        SetServer(event.target.value)
    }

    return (
        <div id="App">
            <div id="result" className="result">{statusText}</div>
            <button className={servers.length === 0 ? "btn" : "hidden"} onClick={linkPlex}>Link Plex</button>
            <select className={servers.length === 0 ? "hidden" : undefined} id="servers" onChange={setServer}>
                {servers.map((server) => (
                    <option key={server}>{server}</option>
                ))}
            </select>
        </div>
    )
}

export default App

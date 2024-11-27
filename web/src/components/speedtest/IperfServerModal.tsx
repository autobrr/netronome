import { Dialog, Transition } from "@headlessui/react";
import { Fragment, useState } from "react";

interface IperfServerModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (name: string) => void;
  title: string;
  message: string;
  confirmText: string;
  confirmStyle?: "danger" | "primary";
  serverDetails?: {
    host: string;
    port: string;
  };
}

export function IperfServerModal({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmText,
  confirmStyle = "primary",
  serverDetails,
}: IperfServerModalProps) {
  const [serverName, setServerName] = useState("");

  return (
    <Transition appear show={isOpen} as={Fragment}>
      <Dialog
        as="div"
        className="relative z-50"
        onClose={() => {
          setServerName("");
          onClose();
        }}
      >
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-black/50 backdrop-blur-sm" />
        </Transition.Child>

        <div className="fixed inset-0 overflow-y-auto">
          <div className="flex min-h-full items-center justify-center p-4">
            <Transition.Child
              as={Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0 scale-95"
              enterTo="opacity-100 scale-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100 scale-100"
              leaveTo="opacity-0 scale-95"
            >
              <Dialog.Panel className="w-full max-w-md transform overflow-hidden rounded-xl bg-gray-850/95 border border-gray-900 p-6 shadow-xl transition-all">
                <Dialog.Title
                  as="h2"
                  className="text-xl font-semibold text-white mb-4"
                >
                  {title}
                </Dialog.Title>

                {serverDetails ? (
                  <div className="space-y-4">
                    <p className="text-gray-400 text-sm">{message}</p>
                    <div className="space-y-2">
                      <label
                        htmlFor="serverName"
                        className="block text-sm font-medium text-gray-400"
                      >
                        Server Name
                      </label>
                      <input
                        type="text"
                        id="serverName"
                        value={serverName}
                        onChange={(e) => setServerName(e.target.value)}
                        placeholder="Enter a name for this server"
                        className="w-full px-4 py-2 bg-gray-800/50 border border-gray-900 text-gray-300 rounded-lg shadow-md focus:outline-none focus:ring-1 focus:ring-inset focus:ring-blue-500/50"
                        autoFocus
                      />
                      <p className="text-sm text-gray-400">
                        Host: {serverDetails.host}:{serverDetails.port}
                      </p>
                    </div>
                  </div>
                ) : (
                  <p className="text-gray-400 text-sm">{message}</p>
                )}

                <div className="mt-6 flex justify-end gap-3">
                  <button
                    type="button"
                    className="px-4 py-2 text-sm text-gray-400 hover:text-gray-200 transition-colors"
                    onClick={() => {
                      setServerName("");
                      onClose();
                    }}
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    disabled={serverDetails && !serverName.trim()}
                    className={`
                      px-4 py-2 
                      rounded-lg 
                      shadow-md
                      transition-colors
                      border
                      text-sm
                      ${
                        serverDetails && !serverName.trim()
                          ? "bg-gray-700 text-gray-400 cursor-not-allowed border-gray-900"
                          : confirmStyle === "danger"
                          ? "bg-red-500 hover:bg-red-600 text-white border-red-600 hover:border-red-700"
                          : "bg-blue-500 hover:bg-blue-600 text-white border-blue-600 hover:border-blue-700"
                      }
                    `}
                    onClick={() => {
                      onConfirm(serverName);
                      setServerName("");
                      onClose();
                    }}
                  >
                    {confirmText}
                  </button>
                </div>
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition>
  );
}

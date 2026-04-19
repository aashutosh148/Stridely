'use client';

import { useEffect, useState } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import Link from 'next/link';
import { useUser } from '../../hooks/useUser';
import { NotificationToaster } from '../../components/notification-toaster';
import {
  Home,
  Calendar,
  MessageSquare,
  Trophy,
  BarChart2,
  Settings,
  Menu,
  X,
  LogOut,
  Activity as ActivityIcon
} from 'lucide-react';

const navigation = [
  { name: 'Dashboard', href: '/dashboard', icon: Home },
  { name: 'Activities', href: '/activities', icon: ActivityIcon },
  { name: 'Plan', href: '/plan', icon: Calendar },
  { name: 'Stats', href: '/stats', icon: BarChart2 },
  { name: 'Race', href: '/race', icon: Trophy },
  { name: 'Chat', href: '/chat', icon: MessageSquare },
  { name: 'Settings', href: '/settings', icon: Settings },
];

export default function AppLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const router = useRouter();
  const pathname = usePathname();
  const { user, isLoading } = useUser();

  useEffect(() => {
    const token = localStorage.getItem('pacer_token');
    if (!token) {
      router.push('/login');
    }
  }, [router]);

  const handleLogout = () => {
    localStorage.removeItem('pacer_token');
    router.push('/login');
  };

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-[#0b0f19]">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-blue-600 border-t-transparent"></div>
      </div>
    );
  }

  const daysToRace = user?.targetRaceId ? 42 : null; 

  return (
    <div className="bg-[#0b0f19]">
      {/* Mobile sidebar */}
      {sidebarOpen && (
        <div className="fixed inset-0 z-50 flex lg:hidden">
          <div className="fixed inset-0 bg-black/80" onClick={() => setSidebarOpen(false)} />
          <div className="relative mr-16 flex w-full max-w-xs flex-1 flex-col bg-[#161b26]">
            <div className="absolute left-full top-0 flex w-16 justify-center pt-5">
              <button type="button" className="-m-2.5 p-2.5" onClick={() => setSidebarOpen(false)}>
                <span className="sr-only">Close sidebar</span>
                <X className="h-6 w-6 text-gray-400" aria-hidden="true" />
              </button>
            </div>
            <div className="flex grow flex-col gap-y-5 overflow-y-auto px-6 pb-4">
              <div className="flex h-16 shrink-0 items-center font-bold text-xl text-gray-100">
                Pacer
              </div>
              <nav className="flex flex-1 flex-col">
                <ul role="list" className="flex flex-1 flex-col gap-y-7">
                  <li>
                    <ul role="list" className="-mx-2 space-y-1">
                      {navigation.map((item) => {
                        const isCurrent = pathname === item.href;
                        return (
                          <li key={item.name}>
                            <Link
                              href={item.href}
                              className={`
                                group flex gap-x-3 rounded p-2 text-sm leading-6 font-medium transition-all
                                ${isCurrent ? 'bg-blue-600 text-white' : 'text-gray-400 hover:bg-[#1e2530] hover:text-gray-200'}
                              `}
                              onClick={() => setSidebarOpen(false)}
                            >
                              <item.icon
                                className={`h-5 w-5 shrink-0 ${isCurrent ? 'text-white' : 'text-gray-500 group-hover:text-gray-300'}`}
                                aria-hidden="true"
                              />
                              {item.name}
                            </Link>
                          </li>
                        );
                      })}
                    </ul>
                  </li>
                  <li className="mt-auto">
                    <button
                      onClick={handleLogout}
                      className="group -mx-2 flex w-full gap-x-3 rounded p-2 text-sm font-medium leading-6 text-gray-400 hover:bg-red-500/10 hover:text-red-400 transition-all border border-transparent hover:border-red-500/30"
                    >
                      <LogOut className="h-5 w-5 shrink-0" aria-hidden="true" />
                      Logout
                    </button>
                  </li>
                </ul>
              </nav>
            </div>
          </div>
        </div>
      )}

      {/* Static sidebar for desktop */}
      <div className="hidden lg:fixed lg:inset-y-0 lg:z-50 lg:flex lg:w-64 lg:flex-col">
        <div className="flex grow flex-col gap-y-5 overflow-y-auto bg-[#161b26] px-6 pb-4 border-r border-gray-800">
          <div className="flex h-16 shrink-0 items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="w-8 h-8 bg-blue-600 rounded flex items-center justify-center">
                <span className="text-white font-bold text-lg">P</span>
              </div>
              <span className="font-bold text-xl text-gray-100 tracking-tight">Pacer</span>
            </div>
            {user?.runner_tier && (
              <span className="inline-flex items-center rounded bg-blue-500/10 px-2 py-1 text-xs font-medium text-blue-400 border border-blue-500/30">
                {user.runner_tier}
              </span>
            )}
          </div>
          
          {daysToRace !== null && (
             <div className="rounded-lg bg-blue-500/10 p-4 border border-blue-500/30">
                <div className="flex items-center">
                  <Trophy className="h-5 w-5 text-blue-400" />
                  <h3 className="ml-2 text-sm font-medium text-gray-300">Next Race</h3>
                </div>
                <div className="mt-2 flex items-baseline gap-x-2">
                  <span className="text-3xl font-bold tracking-tight text-blue-400">{daysToRace}</span>
                  <span className="text-sm font-medium text-gray-400">days to go</span>
                </div>
             </div>
          )}

          <nav className="flex flex-1 flex-col">
            <ul role="list" className="flex flex-1 flex-col gap-y-7">
              <li>
                <ul role="list" className="-mx-2 space-y-1">
                  {navigation.map((item) => {
                    const isCurrent = pathname === item.href;
                    return (
                      <li key={item.name}>
                        <Link
                          href={item.href}
                          className={`
                            group flex gap-x-3 rounded p-2 text-sm leading-6 font-medium transition-all
                            ${isCurrent 
                              ? 'bg-blue-600 text-white' 
                              : 'text-gray-400 hover:bg-[#1e2530] hover:text-gray-200'
                            }
                          `}
                        >
                          <item.icon
                            className={`h-5 w-5 shrink-0 ${isCurrent ? 'text-white' : 'text-gray-500 group-hover:text-gray-300'}`}
                            aria-hidden="true"
                          />
                          {item.name}
                        </Link>
                      </li>
                    );
                  })}
                </ul>
              </li>
              <li className="mt-auto">
                <button
                  onClick={handleLogout}
                  className="group -mx-2 flex w-full gap-x-3 rounded p-2 text-sm font-medium leading-6 text-gray-400 hover:bg-red-500/10 hover:text-red-400 transition-all border border-transparent hover:border-red-500/30"
                >
                  <LogOut className="h-5 w-5 shrink-0" aria-hidden="true" />
                  Logout
                </button>
              </li>
            </ul>
          </nav>
        </div>
      </div>

      <div className="lg:pl-64">
        <div className="sticky top-0 z-40 flex h-16 shrink-0 items-center gap-x-4 border-b border-gray-800 bg-[#161b26] px-4 sm:gap-x-6 sm:px-6 lg:hidden">
          <button type="button" className="-m-2.5 p-2.5 text-gray-400 lg:hidden" onClick={() => setSidebarOpen(true)}>
            <span className="sr-only">Open sidebar</span>
            <Menu className="h-6 w-6" aria-hidden="true" />
          </button>

          <div className="flex flex-1 gap-x-4 self-stretch lg:gap-x-6">
            <div className="flex flex-1 items-center">
              <div className="flex items-center gap-2">
                <div className="w-7 h-7 bg-blue-600 rounded flex items-center justify-center">
                  <span className="text-white font-bold">P</span>
                </div>
                <span className="font-bold text-lg text-gray-100">Pacer</span>
              </div>
            </div>
            <div className="flex items-center gap-x-4 lg:gap-x-6">
               {user?.runner_tier && (
                <span className="inline-flex items-center rounded bg-blue-500/10 px-2 py-1 text-xs font-medium text-blue-400 border border-blue-500/30">
                  {user.runner_tier}
                </span>
               )}
            </div>
          </div>
        </div>

        <main className="bg-[#0b0f19] min-h-screen">
          {children}
        </main>
      </div>
      <NotificationToaster />
    </div>
  );
}
